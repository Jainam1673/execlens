package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// ProcessResult contains execution telemetry and output.
type ProcessResult struct {
	Duration     time.Duration
	PeakMemory   int64 // bytes
	ExitCode     int
	Stdout       string
	Stderr       string
	TimedOut     bool
	Cancelled    bool
	ErrorMessage string
}

// limitedBuffer captures up to maxBytes to avoid unbounded memory usage.
type limitedBuffer struct {
	buf      bytes.Buffer
	maxBytes int
	overflow bool
}

func (l *limitedBuffer) Write(p []byte) (n int, err error) {
	remaining := l.maxBytes - l.buf.Len()
	if remaining <= 0 {
		l.overflow = true
		return len(p), nil
	}
	if len(p) > remaining {
		l.buf.Write(p[:remaining])
		l.overflow = true
		return len(p), nil
	}
	return l.buf.Write(p)
}

func (l *limitedBuffer) String() string {
	s := l.buf.String()
	if l.overflow {
		s += "\n[output truncated: exceeded maximum limit]"
	}
	return s
}

// RunCommand executes a command within an isolated process group and captures execution metrics.
func RunCommand(ctx context.Context, executable string, args []string, workingDir string, env map[string]string, timeout time.Duration) ProcessResult {
	start := time.Now()

	resolvedPath, err := exec.LookPath(executable)
	if err != nil {
		return ProcessResult{
			ExitCode:     -1,
			ErrorMessage: fmt.Sprintf("executable not found: %s", executable),
		}
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, resolvedPath, args...)
	cmd.Dir = workingDir

	// Configure new process group for clean group termination on cancellation/timeout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Controlled environment: standard PATH, TMPDIR, and adapter-declared variables only
	cmdEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"TMPDIR=" + os.TempDir(),
	}
	for k, v := range env {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = cmdEnv

	stdoutBuf := &limitedBuffer{maxBytes: 1024 * 1024} // 1MB limit
	stderrBuf := &limitedBuffer{maxBytes: 1024 * 1024}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if err := cmd.Start(); err != nil {
		return ProcessResult{
			Duration:     time.Since(start),
			ExitCode:     -1,
			ErrorMessage: fmt.Sprintf("failed to start process: %v", err),
		}
	}

	// Background monitor to ensure process group is completely killed if context expires
	done := make(chan struct{})
	go func() {
		select {
		case <-done:
			return
		case <-execCtx.Done():
			if cmd.Process != nil {
				// Send SIGKILL to the entire process group (-pid)
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		}
	}()

	waitErr := cmd.Wait()
	close(done)
	duration := time.Since(start)

	var peakMemoryBytes int64 = 0
	if cmd.ProcessState != nil {
		if rusage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
			// On Linux, rusage.Maxrss is in kilobytes
			peakMemoryBytes = int64(rusage.Maxrss) * 1024
		}
	}

	res := ProcessResult{
		Duration:   duration,
		PeakMemory: peakMemoryBytes,
		Stdout:     stdoutBuf.String(),
		Stderr:     stderrBuf.String(),
	}

	if waitErr != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			res.TimedOut = true
			res.ExitCode = -1
			res.ErrorMessage = fmt.Sprintf("execution timed out after %v", timeout)
		} else if ctx.Err() == context.Canceled {
			res.Cancelled = true
			res.ExitCode = -1
			res.ErrorMessage = "execution cancelled by user"
		} else if exitErr, ok := waitErr.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
			res.ErrorMessage = fmt.Sprintf("process exited with code %d", res.ExitCode)
		} else {
			res.ExitCode = -1
			res.ErrorMessage = waitErr.Error()
		}
	} else {
		res.ExitCode = 0
	}

	return res
}
