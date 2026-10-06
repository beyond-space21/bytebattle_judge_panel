package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/judge/competition/internal/auth"
	"github.com/judge/competition/internal/contest"
	"github.com/judge/competition/internal/judge"
	"github.com/judge/competition/internal/store"
)

func (s *Server) studentLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Secret string `json:"secret"`
		Name   string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Secret = normalizeSecret(req.Secret)
	if len(req.Secret) != 6 || req.Name == "" {
		writeErr(w, http.StatusBadRequest, "secret (6 chars) and name required")
		return
	}
	cfg, err := s.store.GetConfig(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if !cfg.IsActive {
		writeErr(w, http.StatusForbidden, "contest is not active")
		return
	}
	tok, err := auth.RandomToken(24)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token error")
		return
	}
	st, err := s.store.LoginStudent(r.Context(), req.Secret, req.Name, tok, now().Add(s.cfg.SessionTTL))
	if err != nil {
		if err == store.ErrNotFound {
			writeErr(w, http.StatusUnauthorized, "invalid secret")
			return
		}
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "student.login", map[string]any{
		"student_id": st.ID, "name": st.Name, "secret": st.SecretCode,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"token":   tok,
		"student": st,
		"config":  cfg,
		"remaining_seconds": contest.RemainingSeconds(cfg, st, now()),
	})
}

func normalizeSecret(s string) string {
	out := make([]rune, 0, len(s))
	for _, c := range s {
		if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			out = append(out, c)
		}
	}
	return string(out)
}

func (s *Server) studentMe(w http.ResponseWriter, r *http.Request) {
	st := studentFrom(r.Context())
	cfg, err := s.store.GetConfig(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	progress, err := s.store.StudentProgress(r.Context(), st.ID)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"student":           st,
		"config":            cfg,
		"remaining_seconds": contest.RemainingSeconds(cfg, st, now()),
		"expired":           contest.IsExpired(cfg, st, now()),
		"progress":          progress,
	})
}

func (s *Server) studentStart(w http.ResponseWriter, r *http.Request) {
	st := studentFrom(r.Context())
	cfg, err := s.store.GetConfig(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if !cfg.IsActive {
		writeErr(w, http.StatusForbidden, "contest is not active")
		return
	}
	updated, err := s.store.StartStudent(r.Context(), st.ID)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if st.StartedAt == nil {
		s.emit(r.Context(), "student.start", map[string]any{
			"student_id": updated.ID, "name": updated.Name, "started_at": updated.StartedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"student":           updated,
		"remaining_seconds": contest.RemainingSeconds(cfg, updated, now()),
	})
}

func (s *Server) studentListProblems(w http.ResponseWriter, r *http.Request) {
	st := studentFrom(r.Context())
	cfg, err := s.store.GetConfig(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if st.StartedAt == nil {
		writeErr(w, http.StatusForbidden, "start the contest first")
		return
	}
	problems, err := s.store.ListProblems(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	progress, err := s.store.StudentProgress(r.Context(), st.ID)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	progMap := map[int64]store.ProblemProgress{}
	for _, p := range progress {
		progMap[p.ProblemID] = p
	}
	type item struct {
		ID          int64  `json:"id"`
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		Difficulty  string `json:"difficulty"`
		OrderIndex  int    `json:"order_index"`
		BestScore   int    `json:"best_score"`
		MaxScore    int    `json:"max_score"`
		SubmitCount int    `json:"submit_count"`
		SubmitsLeft int    `json:"submits_left"`
	}
	out := make([]item, 0, len(problems))
	for _, p := range problems {
		it := item{ID: p.ID, Slug: p.Slug, Title: p.Title, Difficulty: p.Difficulty, OrderIndex: p.OrderIndex}
		if pp, ok := progMap[p.ID]; ok {
			it.BestScore = pp.BestScore
			it.MaxScore = pp.MaxScore
			it.SubmitCount = pp.SubmitCount
			it.SubmitsLeft = pp.SubmitsLeft
		}
		out = append(out, it)
	}
	_ = cfg
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) studentGetProblem(w http.ResponseWriter, r *http.Request) {
	st := studentFrom(r.Context())
	if st.StartedAt == nil {
		writeErr(w, http.StatusForbidden, "start the contest first")
		return
	}
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
	sampleTrue := true
	samples, err := s.store.ListTestCases(r.Context(), id, &sampleTrue)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	// Strip nothing needed — samples are public
	progress, _ := s.store.StudentProgress(r.Context(), st.ID)
	var pp *store.ProblemProgress
	for i := range progress {
		if progress[i].ProblemID == id {
			pp = &progress[i]
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": p.ID, "slug": p.Slug, "title": p.Title, "statement_md": p.StatementMD,
		"difficulty": p.Difficulty, "starter_code": p.StarterCode,
		"time_limit_ms": p.TimeLimitMS, "memory_limit_kb": p.MemoryLimitKB,
		"samples": samples, "progress": pp,
	})
}

func (s *Server) studentRun(w http.ResponseWriter, r *http.Request) {
	st := studentFrom(r.Context())
	cfg, err := s.store.GetConfig(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if err := contest.CanCompete(cfg, st, now()); err != nil {
		mapStoreErr(w, err)
		return
	}
	if st.StartedAt == nil {
		writeErr(w, http.StatusForbidden, "start the contest first")
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Source      string  `json:"source"`
		CustomInput *string `json:"custom_input"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Source == "" {
		writeErr(w, http.StatusBadRequest, "source required")
		return
	}
	p, err := s.store.GetProblem(r.Context(), id)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if req.CustomInput != nil && strings.TrimSpace(p.ReferenceCode) == "" {
		writeErr(w, http.StatusBadRequest, "custom input is unavailable: no official solution configured for this problem")
		return
	}

	sub := &store.Submission{
		StudentID: st.ID, ProblemID: id, Kind: "run", Source: req.Source, Status: "pending",
	}
	sub, err = s.store.CreateSubmission(r.Context(), sub)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "submission.created", map[string]any{
		"id": sub.ID, "kind": "run", "student_id": st.ID, "name": st.Name, "problem_id": id, "problem_title": p.Title,
	})

	var result any
	var status string
	var score, maxScore int
	var detail []byte

	if req.CustomInput != nil {
		cr, err := s.judge.RunCustom(r.Context(), req.Source, p.ReferenceCode, *req.CustomInput, p.TimeLimitMS, p.MemoryLimitKB)
		if err != nil {
			_, _ = s.store.UpdateSubmissionResult(r.Context(), sub.ID, "error", 0, 0, mustRaw(map[string]string{"error": err.Error()}))
			writeErr(w, http.StatusBadGateway, "judge error: "+err.Error())
			return
		}
		status = cr.Status
		result = cr
		detail = mustRaw(cr)
	} else {
		sampleTrue := true
		samples, err := s.store.ListTestCases(r.Context(), id, &sampleTrue)
		if err != nil {
			mapStoreErr(w, err)
			return
		}
		cases := toJudgeCases(samples)
		rr, err := s.judge.Evaluate(r.Context(), req.Source, cases, p.TimeLimitMS, p.MemoryLimitKB, true)
		if err != nil {
			_, _ = s.store.UpdateSubmissionResult(r.Context(), sub.ID, "error", 0, 0, mustRaw(map[string]string{"error": err.Error()}))
			writeErr(w, http.StatusBadGateway, "judge error: "+err.Error())
			return
		}
		status = rr.Status
		score, maxScore = rr.Score, rr.MaxScore
		result = rr
		detail = mustRaw(rr.Cases)
	}

	updated, err := s.store.UpdateSubmissionResult(r.Context(), sub.ID, status, score, maxScore, detail)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "submission.finished", map[string]any{
		"id": updated.ID, "kind": "run", "status": status, "score": score, "max_score": maxScore,
		"student_id": st.ID, "name": st.Name, "problem_id": id, "problem_title": p.Title,
	})
	writeJSON(w, http.StatusOK, map[string]any{"submission": updated, "result": result})
}

func (s *Server) studentSubmit(w http.ResponseWriter, r *http.Request) {
	st := studentFrom(r.Context())
	cfg, err := s.store.GetConfig(r.Context())
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if st.StartedAt == nil {
		writeErr(w, http.StatusForbidden, "start the contest first")
		return
	}
	if err := contest.CanCompete(cfg, st, now()); err != nil {
		mapStoreErr(w, err)
		return
	}
	id, err := parseID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Source string `json:"source"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Source == "" {
		writeErr(w, http.StatusBadRequest, "source required")
		return
	}
	count, err := s.store.CountSubmits(r.Context(), st.ID, id)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if count >= 3 {
		writeErr(w, http.StatusForbidden, "submission limit reached (3)")
		return
	}
	p, err := s.store.GetProblem(r.Context(), id)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	sampleFalse := false
	hidden, err := s.store.ListTestCases(r.Context(), id, &sampleFalse)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	if len(hidden) == 0 {
		writeErr(w, http.StatusBadRequest, "no hidden test cases configured")
		return
	}

	sub := &store.Submission{
		StudentID: st.ID, ProblemID: id, Kind: "submit", Source: req.Source, Status: "pending",
	}
	sub, err = s.store.CreateSubmission(r.Context(), sub)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "submission.created", map[string]any{
		"id": sub.ID, "kind": "submit", "student_id": st.ID, "name": st.Name, "problem_id": id, "problem_title": p.Title,
	})

	cases := toJudgeCases(hidden)
	rr, err := s.judge.Evaluate(r.Context(), req.Source, cases, p.TimeLimitMS, p.MemoryLimitKB, false)
	if err != nil {
		_, _ = s.store.UpdateSubmissionResult(r.Context(), sub.ID, "error", 0, 0, mustRaw(map[string]string{"error": err.Error()}))
		writeErr(w, http.StatusBadGateway, "judge error: "+err.Error())
		return
	}

	// Sanitize: no hidden I/O for students
	safeCases := make([]map[string]any, 0, len(rr.Cases))
	for i, c := range rr.Cases {
		safeCases = append(safeCases, map[string]any{
			"index": i + 1, "passed": c.Passed, "points": c.Points, "status": c.Status,
			"time": c.Time, "memory": c.Memory,
		})
	}
	detail := mustRaw(safeCases)
	updated, err := s.store.UpdateSubmissionResult(r.Context(), sub.ID, rr.Status, rr.Score, rr.MaxScore, detail)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	s.emit(r.Context(), "submission.finished", map[string]any{
		"id": updated.ID, "kind": "submit", "status": rr.Status, "score": rr.Score, "max_score": rr.MaxScore,
		"student_id": st.ID, "name": st.Name, "problem_id": id, "problem_title": p.Title,
	})
	// Refresh leaderboard broadcast
	if lb, err := s.store.Leaderboard(r.Context()); err == nil {
		s.hub.Broadcast("leaderboard.updated", lb)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"submission": updated,
		"result": map[string]any{
			"status": rr.Status, "score": rr.Score, "max_score": rr.MaxScore, "cases": safeCases,
		},
	})
}

func (s *Server) studentSubmissions(w http.ResponseWriter, r *http.Request) {
	st := studentFrom(r.Context())
	subs, err := s.store.ListSubmissionsByStudent(r.Context(), st.ID)
	if err != nil {
		mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func toJudgeCases(tcs []store.TestCase) []judge.CaseInput {
	out := make([]judge.CaseInput, 0, len(tcs))
	for _, tc := range tcs {
		out = append(out, judge.CaseInput{
			ID: tc.ID, Input: tc.Input, Expected: tc.ExpectedOutput, Points: tc.Points, IsSample: tc.IsSample,
		})
	}
	return out
}

func mustRaw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
