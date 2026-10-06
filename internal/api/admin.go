package api

import (
	"net/http"

	"github.com/judge/competition/internal/auth"
	"github.com/judge/competition/internal/store"
)

func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	admin, err := s.store.GetAdminByUsername(r.Context(), req.Username)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !auth.CheckPassword(admin.PasswordHash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	tok, err := auth.IssueAdminJWT(s.cfg.JWTSecret, admin.ID, admin.Username, s.cfg.SessionTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "username": admin.Username})
}

func (s *Server) adminGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetConfig(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) adminUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title           string `json:"title"`
		DurationMinutes int    `json:"duration_minutes"`
		IsActive        bool   `json:"is_active"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Title == "" || req.DurationMinutes <= 0 {
		writeErr(w, http.StatusBadRequest, "title and positive duration required")
		return
	}
	cfg, err := s.store.UpdateConfig(r.Context(), req.Title, req.DurationMinutes, req.IsActive)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "config.updated", cfg)
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) adminGenerateSecrets(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Count int `json:"count"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Count <= 0 {
		writeErr(w, http.StatusBadRequest, "count > 0 required")
		return
	}
	if req.Count > 500 {
		writeErr(w, http.StatusBadRequest, "max 500 secrets per request")
		return
	}
	codes := make([]string, 0, req.Count)
	seen := map[string]struct{}{}
	for len(codes) < req.Count {
		code, err := auth.GenerateSecretCode()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "generate error")
			return
		}
		if _, ok := seen[code]; ok {
			continue
		}
		exists, err := s.store.SecretExists(r.Context(), code)
		if err != nil {
			mapStoreErr(w, err)
			return
		}
		if exists {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	secs, err := s.store.CreateSecrets(r.Context(), codes)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "secrets.generated", map[string]any{"count": len(secs)})
	writeJSON(w, http.StatusOK, secs)
}

func (s *Server) adminListSecrets(w http.ResponseWriter, r *http.Request) {
	secs, err := s.store.ListSecrets(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, secs)
}

func (s *Server) adminDeleteAllSecrets(w http.ResponseWriter, r *http.Request) {
	secretsDeleted, studentsDeleted, err := s.store.DeleteAllSecrets(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	payload := map[string]any{
		"secrets_deleted":  secretsDeleted,
		"students_deleted": studentsDeleted,
	}
	s.emit(r.Context(), "secrets.deleted_all", payload)
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) adminResetContest(w http.ResponseWriter, r *http.Request) {
	students, secrets, submissions, events, err := s.store.ResetContest(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	payload := map[string]any{
		"students_deleted":    students,
		"secrets_deleted":     secrets,
		"submissions_deleted": submissions,
		"events_deleted":      events,
	}
	s.emit(r.Context(), "contest.reset", payload)
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) adminListProblems(w http.ResponseWriter, r *http.Request) {
	problems, err := s.store.ListProblems(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, problems)
}

func (s *Server) adminGetProblem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := s.store.GetProblem(r.Context(), id)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) adminCreateProblem(w http.ResponseWriter, r *http.Request) {
	var p store.Problem
	if err := decodeJSON(r, &p); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if p.Slug == "" || p.Title == "" {
		writeErr(w, http.StatusBadRequest, "slug and title required")
		return
	}
	if p.Difficulty == "" {
		p.Difficulty = "medium"
	}
	if p.StarterCode == "" {
		p.StarterCode = "def solve():\n    pass\n"
	}
	if p.TimeLimitMS == 0 {
		p.TimeLimitMS = 2000
	}
	if p.MemoryLimitKB == 0 {
		p.MemoryLimitKB = 256000
	}
	created, err := s.store.CreateProblem(r.Context(), &p)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "problem.created", created)
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) adminUpdateProblem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p store.Problem
	if err := decodeJSON(r, &p); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	p.ID = id
	updated, err := s.store.UpdateProblem(r.Context(), &p)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "problem.updated", updated)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) adminDeleteProblem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteProblem(r.Context(), id); err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "problem.deleted", map[string]any{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) adminReplaceTestCases(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		TestCases []store.TestCase `json:"test_cases"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	tcs, err := s.store.ReplaceTestCases(r.Context(), id, req.TestCases)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "problem.test_cases_updated", map[string]any{"problem_id": id, "count": len(tcs)})
	writeJSON(w, http.StatusOK, tcs)
}

func (s *Server) adminListStudents(w http.ResponseWriter, r *http.Request) {
	students, err := s.store.ListStudents(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, students)
}

func (s *Server) adminListSubmissions(w http.ResponseWriter, r *http.Request) {
	subs, err := s.store.ListRecentSubmissions(r.Context(), 100)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func (s *Server) adminLeaderboard(w http.ResponseWriter, r *http.Request) {
	lb, err := s.store.Leaderboard(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, lb)
}

func (s *Server) adminActivity(w http.ResponseWriter, r *http.Request) {
	evs, err := s.store.ListAudit(r.Context(), 100)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, evs)
}

func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.Stats(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) adminDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.DashboardData(r.Context(), now())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) adminWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.hub.Add(conn)
	defer s.hub.Remove(conn)
	// Keep connection alive; ignore client messages
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
