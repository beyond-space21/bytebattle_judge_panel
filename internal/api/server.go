package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"

	"github.com/judge/competition/internal/auth"
	"github.com/judge/competition/internal/config"
	"github.com/judge/competition/internal/judge"
	"github.com/judge/competition/internal/store"
)

type Server struct {
	cfg    config.Config
	store  *store.Store
	judge  *judge.Client
	hub    *Hub
	router chi.Router
	static fs.FS
}

func New(cfg config.Config, st *store.Store, j *judge.Client, static fs.FS) *Server {
	s := &Server{
		cfg:    cfg,
		store:  st,
		judge:  j,
		hub:    NewHub(),
		static: static,
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.router }

func (s *Server) routes() {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.cfg.CORSOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Student-Token"},
		AllowCredentials: s.cfg.CORSOrigin != "*",
		MaxAge:           300,
	}))

	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/student", func(r chi.Router) {
		r.Post("/login", s.studentLogin)
		r.Group(func(r chi.Router) {
			r.Use(s.studentAuth)
			r.Get("/me", s.studentMe)
			r.Post("/start", s.studentStart)
			r.Get("/problems", s.studentListProblems)
			r.Get("/problems/{id}", s.studentGetProblem)
			r.Post("/problems/{id}/run", s.studentRun)
			r.Post("/problems/{id}/submit", s.studentSubmit)
			r.Get("/submissions", s.studentSubmissions)
		})
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Post("/login", s.adminLogin)
		r.Group(func(r chi.Router) {
			r.Use(s.adminAuth)
			r.Get("/config", s.adminGetConfig)
			r.Put("/config", s.adminUpdateConfig)
			r.Post("/secrets/generate", s.adminGenerateSecrets)
			r.Get("/secrets", s.adminListSecrets)
			r.Delete("/secrets", s.adminDeleteAllSecrets)
			r.Post("/reset", s.adminResetContest)
			r.Get("/problems", s.adminListProblems)
			r.Post("/problems", s.adminCreateProblem)
			r.Get("/problems/{id}", s.adminGetProblem)
			r.Put("/problems/{id}", s.adminUpdateProblem)
			r.Delete("/problems/{id}", s.adminDeleteProblem)
			r.Put("/problems/{id}/test-cases", s.adminReplaceTestCases)
			r.Get("/students", s.adminListStudents)
			r.Get("/submissions", s.adminListSubmissions)
			r.Get("/leaderboard", s.adminLeaderboard)
			r.Get("/activity", s.adminActivity)
			r.Get("/stats", s.adminStats)
			r.Get("/dashboard", s.adminDashboard)
			r.Get("/ws", s.adminWS)
		})
	})

	if s.static != nil {
		fileServer(r, s.static)
	}

	s.router = r
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (s *Server) studentAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get("X-Student-Token")
		if tok == "" {
			authz := r.Header.Get("Authorization")
			if strings.HasPrefix(authz, "Bearer ") {
				tok = strings.TrimPrefix(authz, "Bearer ")
			}
		}
		if tok == "" {
			writeErr(w, http.StatusUnauthorized, "missing student token")
			return
		}
		st, err := s.store.StudentBySession(r.Context(), tok)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid session")
			return
		}
		ctx := context.WithValue(r.Context(), ctxStudentKey{}, st)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			// WS can pass token as query
			if t := r.URL.Query().Get("token"); t != "" {
				authz = "Bearer " + t
			} else {
				writeErr(w, http.StatusUnauthorized, "missing admin token")
				return
			}
		}
		claims, err := auth.ParseAdminJWT(s.cfg.JWTSecret, strings.TrimPrefix(authz, "Bearer "))
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid admin token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxAdminKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type ctxStudentKey struct{}
type ctxAdminKey struct{}

func studentFrom(ctx context.Context) *store.Student {
	st, _ := ctx.Value(ctxStudentKey{}).(*store.Student)
	return st
}

func isStaticAsset(path string) bool {
	if path == "" {
		return false
	}
	i := strings.LastIndex(path, ".")
	if i < 0 || i == len(path)-1 {
		return false
	}
	ext := strings.ToLower(path[i+1:])
	switch ext {
	case "js", "css", "map", "png", "jpg", "jpeg", "gif", "svg", "webp", "ico", "woff", "woff2", "ttf":
		return true
	default:
		return false
	}
}

func fileServer(r chi.Router, static fs.FS) {
	serveIndex := func(w http.ResponseWriter) {
		data, err := fs.ReadFile(static, "index.html")
		if err != nil {
			http.Error(w, "ui not built", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		_, _ = w.Write(data)
	}
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" || path == "admin" || strings.HasPrefix(path, "admin/") {
			serveIndex(w)
			return
		}
		f, err := static.Open(path)
		if err != nil {
			if isStaticAsset(path) {
				http.NotFound(w, req)
				return
			}
			serveIndex(w)
			return
		}
		stat, err := f.Stat()
		f.Close()
		if err != nil || stat.IsDir() {
			if isStaticAsset(path) {
				http.NotFound(w, req)
				return
			}
			serveIndex(w)
			return
		}
		if isStaticAsset(path) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.FileServer(http.FS(static)).ServeHTTP(w, req)
	})
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func mapStoreErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, store.ErrExpired):
		writeErr(w, http.StatusForbidden, "time expired")
	case errors.Is(err, store.ErrInactive):
		writeErr(w, http.StatusForbidden, "contest is not active")
	case errors.Is(err, store.ErrLimitExceeded):
		writeErr(w, http.StatusForbidden, "submission limit reached")
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
	return true
}

func (s *Server) emit(ctx context.Context, eventType string, payload any) {
	ev, err := s.store.AddAudit(ctx, eventType, payload)
	if err == nil {
		s.hub.Broadcast(eventType, map[string]any{
			"event":   ev,
			"payload": payload,
		})
	} else {
		s.hub.Broadcast(eventType, payload)
	}
}

func now() time.Time { return time.Now().UTC() }
