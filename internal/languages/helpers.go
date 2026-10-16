package languages

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// checkToolVersion verifies that the tool exists in PATH and runs a version command.
func checkToolVersion(ctx context.Context, tool string, args ...string) (bool, string, string) {
	path, err := exec.LookPath(tool)
	if err != nil {
		return false, "", fmt.Sprintf("Executable '%s' not found in system PATH", tool)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		out := strings.TrimSpace(stdout.String() + " " + stderr.String())
		return false, "", fmt.Sprintf("Failed to run %s %v: %v (%s)", tool, args, err, out)
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		out = strings.TrimSpace(stderr.String())
	}
	// Return first line of output
	lines := strings.Split(out, "\n")
	if len(lines) > 0 {
		return true, strings.TrimSpace(lines[0]), ""
	}
	return true, "Installed", ""
}
