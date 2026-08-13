package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/chenq7an/gstor/common/benchmark"
)

func TestBenchmarkRunCommandWritesJSONToStdout(t *testing.T) {
	original := runBenchmark
	runBenchmark = func(ctx context.Context, options benchmark.RunOptions) (benchmark.RunOutput, error) {
		return benchmark.RunOutput{
			Profile: benchmark.ProfileSummary{Name: "short", Version: "v1", Hash: "abc", BaselineCandidate: false},
			Results: []benchmark.DiskBenchmarkResult{{Disk: benchmark.DiskTarget{Name: "sdb", DevicePath: "/dev/sdb"}}},
		}, nil
	}
	defer func() { runBenchmark = original }()

	cmd := newBenchmarkCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"run", "--profile", "short", "--disk", "sdb"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("benchmark command returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for fake run", stderr.String())
	}
	var output benchmark.RunOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not JSON: %v; %q", err, stdout.String())
	}
	if len(output.Results) != 1 || output.Results[0].Disk.Name != "sdb" {
		t.Fatalf("unexpected output: %+v", output)
	}
}
