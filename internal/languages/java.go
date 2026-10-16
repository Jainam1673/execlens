package languages

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type JavaAdapter struct{}

func NewJavaAdapter() *JavaAdapter {
	return &JavaAdapter{}
}

func (j *JavaAdapter) ID() string {
	return "java"
}

func (j *JavaAdapter) DisplayName() string {
	return "Java (OpenJDK JVM)"
}

func (j *JavaAdapter) RequiredTools() []string {
	return []string{"javac", "java"}
}

func (j *JavaAdapter) DetectAvailability(ctx context.Context) (bool, string, string) {
	avail, ver, missing := checkToolVersion(ctx, "javac", "-version")
	if !avail {
		return false, "", missing
	}
	avail2, _, missing2 := checkToolVersion(ctx, "java", "-version")
	if !avail2 {
		return false, "", missing2
	}
	return true, ver, ""
}

func (j *JavaAdapter) IsCompiled() bool {
	return true
}

func (j *JavaAdapter) SourceSubdir() string {
	return "java"
}

// className converts "sieve" -> "Sieve", "json_proc" -> "JsonProc"
func (j *JavaAdapter) className(workloadID string) string {
	parts := strings.Split(workloadID, "_")
	var res string
	for _, p := range parts {
		if len(p) > 0 {
			res += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return res
}

func (j *JavaAdapter) BuildCommand(workloadID, sourceDir, targetBinPath string) (string, []string, string) {
	clsName := j.className(workloadID)
	srcFile := filepath.Join(sourceDir, fmt.Sprintf("%s.java", clsName))
	// targetBinPath is directory for compiled .class files
	args := []string{
		"-d", targetBinPath,
		srcFile,
	}
	return "javac", args, sourceDir
}

func (j *JavaAdapter) ExecutionCommand(workloadID, targetBinPath string, workloadArgs []string) (string, []string, string) {
	clsName := j.className(workloadID)
	args := append([]string{
		"-XX:+TieredCompilation",
		"-cp", targetBinPath,
		clsName,
	}, workloadArgs...)
	return "java", args, targetBinPath
}

func (j *JavaAdapter) BuildTimeout() time.Duration {
	return 30 * time.Second
}

func (j *JavaAdapter) ExecutionTimeout() time.Duration {
	return 60 * time.Second
}

func (j *JavaAdapter) DefaultWarmupIterations() int {
	return 2
}

func (j *JavaAdapter) WarmupPolicyDescription() string {
	return "JVM utilizes HotSpot tiered compilation (C1 & C2 JIT); 2 warm-up iterations allow bytecode profiling and optimization before steady-state measurement."
}

func (j *JavaAdapter) EnvironmentVariables() map[string]string {
	return map[string]string{}
}
