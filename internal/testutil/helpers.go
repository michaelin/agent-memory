// Package testutil provides shared test helpers for agent-memory tests.
package testutil

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// BuildBinary builds the agent-memory binary and returns its path.
func BuildBinary(t testing.TB) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "agent-memory")

	moduleRoot := findModuleRoot()
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/agent-memory")
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("BuildBinary failed: %v\n%s", err, out)
	}
	return binPath
}

// RunBinary runs the binary with args and returns stdout, stderr, and exit code.
func RunBinary(binPath string, args ...string) (stdout, stderr string, exitCode int) {
	cmd := exec.Command(binPath, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return
}

// MustStat asserts a path exists and returns its FileInfo.
func MustStat(t testing.TB, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
	return info
}

func findModuleRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find go.mod")
		}
		dir = parent
	}
}
