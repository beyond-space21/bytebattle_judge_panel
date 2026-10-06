package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	languageID int
	backend    string // auto | judge0 | local
	local      LocalRunner
}

func New(baseURL string, languageID int) *Client {
	backend := os.Getenv("JUDGE_BACKEND")
	if backend == "" {
		backend = "auto"
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 60 * time.Second},
		languageID: languageID,
		backend:    backend,
	}
}

type CaseResult struct {
	TestCaseID int64  `json:"test_case_id"`
	Passed     bool   `json:"passed"`
	Points     int    `json:"points"`
	Status     string `json:"status"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	Time       string `json:"time,omitempty"`
	Memory     int    `json:"memory,omitempty"`
	IsSample   bool   `json:"is_sample"`
	Expected   string `json:"expected,omitempty"`
	Input      string `json:"input,omitempty"`
}

type CaseInput struct {
	ID       int64
	Input    string
	Expected string
	Points   int
	IsSample bool
}

type RunResult struct {
	Status   string       `json:"status"`
	Score    int          `json:"score"`
	MaxScore int          `json:"max_score"`
	Cases    []CaseResult `json:"cases"`
}

type submissionReq struct {
	SourceCode     string  `json:"source_code"`
	LanguageID     int     `json:"language_id"`
	Stdin          string  `json:"stdin"`
	ExpectedOutput string  `json:"expected_output,omitempty"`
	CPUTimeLimit   float64 `json:"cpu_time_limit,omitempty"`
	MemoryLimit    int     `json:"memory_limit,omitempty"`
}

type submissionResp struct {
	Token  string `json:"token"`
	Status struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"status"`
	Stdout        *string `json:"stdout"`
	Stderr        *string `json:"stderr"`
	CompileOutput *string `json:"compile_output"`
	Message       *string `json:"message"`
	Time          *string `json:"time"`
	Memory        *int    `json:"memory"`
}

func (c *Client) Evaluate(ctx context.Context, source string, cases []CaseInput, timeLimitMS, memoryKB int, includeIO bool) (*RunResult, error) {
	result := &RunResult{Cases: make([]CaseResult, 0, len(cases))}
	allAC := true
	for _, tc := range cases {
		sr, err := c.execute(ctx, source, tc.Input, tc.Expected, timeLimitMS, memoryKB)
		if err != nil {
			return nil, err
		}
		cr := CaseResult{
			TestCaseID: tc.ID,
			Points:     0,
			Status:     statusLabel(sr.Status.ID, sr.Status.Description),
			Stdout:     deref(sr.Stdout),
			Stderr:     combineErr(sr.Stderr, sr.CompileOutput, sr.Message),
			IsSample:   tc.IsSample,
		}
		if sr.Time != nil {
			cr.Time = *sr.Time
		}
		if sr.Memory != nil {
			cr.Memory = *sr.Memory
		}
		if includeIO {
			cr.Input = tc.Input
			cr.Expected = tc.Expected
		}
		passed := sr.Status.ID == 3
		cr.Passed = passed
		if passed {
			cr.Points = tc.Points
			result.Score += tc.Points
		} else {
			allAC = false
		}
		result.MaxScore += tc.Points
		result.Cases = append(result.Cases, cr)
	}
	if len(cases) == 0 {
		result.Status = "No Tests"
	} else if allAC {
		result.Status = "Accepted"
	} else if result.Score > 0 {
		result.Status = "Partial"
	} else {
		result.Status = result.Cases[0].Status
	}
	return result, nil
}

func (c *Client) RunCustom(ctx context.Context, source, reference, stdin string, timeLimitMS, memoryKB int) (*CaseResult, error) {
	student, err := c.execute(ctx, source, stdin, "", timeLimitMS, memoryKB)
	if err != nil {
		return nil, err
	}
	cr := &CaseResult{
		Stdout: deref(student.Stdout),
		Stderr: combineErr(student.Stderr, student.CompileOutput, student.Message),
		Input:  stdin,
	}
	if student.Time != nil {
		cr.Time = *student.Time
	}
	if student.Memory != nil {
		cr.Memory = *student.Memory
	}

	// Student failed to run — report that first.
	if student.Status.ID != 3 {
		cr.Passed = false
		cr.Status = statusLabel(student.Status.ID, student.Status.Description)
		return cr, nil
	}

	if strings.TrimSpace(reference) == "" {
		cr.Passed = false
		cr.Status = "Finished"
		cr.Stderr = strings.TrimSpace(strings.Join([]string{cr.Stderr, "No official solution configured for custom-input check"}, "\n"))
		return cr, nil
	}

	ref, err := c.execute(ctx, reference, stdin, "", timeLimitMS, memoryKB)
	if err != nil {
		return nil, err
	}
	if ref.Status.ID != 3 {
		cr.Passed = false
		cr.Status = "Internal Error"
		cr.Stderr = strings.TrimSpace(strings.Join([]string{
			cr.Stderr,
			"Official solution failed: " + statusLabel(ref.Status.ID, ref.Status.Description),
			combineErr(ref.Stderr, ref.CompileOutput, ref.Message),
		}, "\n"))
		return cr, nil
	}

	expected := deref(ref.Stdout)
	cr.Expected = expected
	if outputsMatch(cr.Stdout, expected) {
		cr.Passed = true
		cr.Status = "Accepted"
	} else {
		cr.Passed = false
		cr.Status = "Wrong Answer"
	}
	return cr, nil
}

func (c *Client) execute(ctx context.Context, source, stdin, expected string, timeLimitMS, memoryKB int) (*submissionResp, error) {
	cpuLimit := float64(timeLimitMS) / 1000.0
	if cpuLimit <= 0 {
		cpuLimit = 2
	}

	switch c.backend {
	case "local":
		return c.runLocal(ctx, source, stdin, expected, timeLimitMS)
	case "judge0":
		return c.runJudge0(ctx, source, stdin, expected, cpuLimit, memoryKB)
	default: // auto
		sr, err := c.runJudge0(ctx, source, stdin, expected, cpuLimit, memoryKB)
		if err != nil || sr.Status.ID == 13 {
			log.Printf("judge0 unavailable or internal error; falling back to local python runner")
			return c.runLocal(ctx, source, stdin, expected, timeLimitMS)
		}
		return sr, nil
	}
}

func (c *Client) runLocal(ctx context.Context, source, stdin, expected string, timeLimitMS int) (*submissionResp, error) {
	sr, err := c.local.Run(ctx, source, stdin, timeLimitMS)
	if err != nil {
		return nil, err
	}
	// If process ran successfully, compare outputs when expected is set
	if sr.Status.ID == 3 && expected != "" {
		if !outputsMatch(deref(sr.Stdout), expected) {
			sr.Status.ID = 4
			sr.Status.Description = "Wrong Answer"
		}
	}
	return sr, nil
}

func (c *Client) runJudge0(ctx context.Context, source, stdin, expected string, cpuLimit float64, memoryKB int) (*submissionResp, error) {
	body := submissionReq{
		SourceCode:   source,
		LanguageID:   c.languageID,
		Stdin:        stdin,
		CPUTimeLimit: cpuLimit,
		MemoryLimit:  memoryKB,
	}
	if expected != "" {
		body.ExpectedOutput = expected
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/submissions?base64_encoded=false&wait=true", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("judge0 request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		var created submissionResp
		if err := json.Unmarshal(data, &created); err == nil && created.Token != "" {
			return c.poll(ctx, created.Token)
		}
		return nil, fmt.Errorf("judge0 status %d: %s", resp.StatusCode, string(data))
	}
	var out submissionResp
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out.Status.ID == 1 || out.Status.ID == 2 {
		if out.Token != "" {
			return c.poll(ctx, out.Token)
		}
	}
	return &out, nil
}

func (c *Client) poll(ctx context.Context, token string) (*submissionResp, error) {
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/submissions/"+token+"?base64_encoded=false", nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var out submissionResp
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		if out.Status.ID > 2 {
			return &out, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("judge0 timeout waiting for %s", token)
}

func statusLabel(id int, desc string) string {
	switch id {
	case 3:
		return "Accepted"
	case 4:
		return "Wrong Answer"
	case 5:
		return "Time Limit Exceeded"
	case 6:
		return "Compilation Error"
	case 7, 8, 9, 10, 11, 12:
		return "Runtime Error"
	case 13:
		return "Internal Error"
	case 14:
		return "Exec Format Error"
	default:
		if desc != "" {
			return desc
		}
		return fmt.Sprintf("Status %d", id)
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func combineErr(parts ...*string) string {
	out := []string{}
	for _, p := range parts {
		if p != nil && *p != "" {
			out = append(out, *p)
		}
	}
	return strings.Join(out, "\n")
}
