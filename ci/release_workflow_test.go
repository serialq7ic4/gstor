package ci_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseWorkflowBuildsStaticLinuxBinaries(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}

	repoRoot := filepath.Dir(filepath.Dir(filename))
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "release.yml")
	workflowBytes, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	workflow := string(workflowBytes)

	requiredSnippets := []string{
		"CGO_ENABLED=0 GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} go build",
		"ldd \"$package_dir/gstor\" 2>&1 | tee /tmp/gstor-ldd.txt",
		"not a dynamic executable",
		"gh release upload",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(workflow, snippet) {
			t.Fatalf("release workflow is missing %q", snippet)
		}
	}
}
