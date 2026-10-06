package judge

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LocalRunner executes Python locally with a wall-clock timeout.
// Used as fallback when Judge0 isolate fails (e.g. cgroup v2 hosts).
type LocalRunner struct{}

func (LocalRunner) Run(ctx context.Context, source, stdin string, timeLimitMS int) (*submissionResp, error) {
	dir, err := os.MkdirTemp("", "judge-py-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	script := filepath.Join(dir, "main.py")
	if err := os.WriteFile(script, []byte(source), 0o600); err != nil {
		return nil, err
	}

	timeout := time.Duration(timeLimitMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	// small grace for interpreter startup
	timeout += 500 * time.Millisecond

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "python3", script)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	elapsed := time.Since(start).Seconds()
	timeStr := fmt.Sprintf("%.3f", elapsed)

	out := &submissionResp{}
	out.Time = &timeStr
	sout := stdout.String()
	serr := stderr.String()
	out.Stdout = &sout
	out.Stderr = &serr

	if runCtx.Err() == context.DeadlineExceeded {
		out.Status.ID = 5
		out.Status.Description = "Time Limit Exceeded"
		return out, nil
	}
	if err != nil {
		out.Status.ID = 11
		out.Status.Description = "Runtime Error"
		return out, nil
	}
	out.Status.ID = 3
	out.Status.Description = "Finished"
	return out, nil
}

func normalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimRight(s, "\n")
	return s
}

func outputsMatch(got, expected string) bool {
	return normalizeOutput(got) == normalizeOutput(expected)
}
