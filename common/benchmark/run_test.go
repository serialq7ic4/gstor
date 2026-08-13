package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeFIORunner struct {
	calls int
	cases []FIOCaseResult
	err   error
}

func (r *fakeFIORunner) RunFIO(ctx context.Context, profile Profile, disk DiskTarget) ([]FIOCaseResult, error) {
	r.calls++
	return r.cases, r.err
}

type fakeReporter struct {
	posted []DiskBenchmarkResult
	err    error
}

func (r *fakeReporter) Post(ctx context.Context, url string, result DiskBenchmarkResult) error {
	r.posted = append(r.posted, result)
	return r.err
}

func completeCaseResults(t *testing.T, profile Profile) []FIOCaseResult {
	t.Helper()
	cases := make([]FIOCaseResult, 0, len(profile.Cases))
	for _, c := range profile.Cases {
		cases = append(cases, FIOCaseResult{
			CaseID:    c.ID,
			CaseOrder: c.Order,
			Parameters: map[string]any{
				"rw": c.RW,
				"bs": c.BlockSize,
			},
			Metrics: map[string]any{"read_iops": float64(c.Order)},
		})
	}
	return cases
}

func TestRunBenchmarkFailsRequirementsBeforeFIO(t *testing.T) {
	fio := &fakeFIORunner{}
	_, err := RunBenchmark(context.Background(), RunOptions{
		ProfileName: "short",
		Probe: fakeProbe{
			paths: map[string]string{"fio": "/usr/bin/fio"},
			euid:  1000,
		},
		Discoverer: func(context.Context, []string) ([]DiskTarget, []SkippedDisk, error) {
			t.Fatal("discoverer should not run when requirements fail")
			return nil, nil, nil
		},
		FIORunner: fio,
	})
	if err == nil {
		t.Fatal("RunBenchmark should fail when requirements are missing")
	}
	if fio.calls != 0 {
		t.Fatalf("fio calls = %d, want 0", fio.calls)
	}
}

func TestRunBenchmarkWritesOutputAndReportsCompletedDisks(t *testing.T) {
	profile, err := SelectProfile("short")
	if err != nil {
		t.Fatalf("SelectProfile(short): %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "benchmark.json")
	fio := &fakeFIORunner{cases: completeCaseResults(t, profile)}
	reporter := &fakeReporter{}

	output, err := RunBenchmark(context.Background(), RunOptions{
		ProfileName: "short",
		OutputPath:  outputPath,
		ReportURL:   "https://collector.example/api/v1/benchmark/results",
		Probe: fakeProbe{
			paths: map[string]string{
				"fio": "/usr/bin/fio", "lsblk": "/usr/bin/lsblk", "blkid": "/usr/sbin/blkid",
				"findmnt": "/usr/bin/findmnt", "fuser": "/usr/bin/fuser", "smartctl": "/usr/sbin/smartctl",
			},
			euid: 0,
		},
		Discoverer: func(context.Context, []string) ([]DiskTarget, []SkippedDisk, error) {
			return []DiskTarget{{Name: "sdb", DevicePath: "/dev/sdb", MediaType: "SSD", InterfaceType: "SATA"}}, nil, nil
		},
		FIORunner: fio,
		Reporter:  reporter,
		ServerInfoProvider: func(context.Context) (ServerInfo, error) {
			return ServerInfo{Hostname: "host-a", SN: "SN123", PrimaryIP: "10.0.0.1"}, nil
		},
		LockPath: filepath.Join(t.TempDir(), "gstor.lock"),
	})
	if err != nil {
		t.Fatalf("RunBenchmark returned error: %v", err)
	}
	if len(output.Results) != 1 {
		t.Fatalf("result count = %d, want 1", len(output.Results))
	}
	if len(reporter.posted) != 1 {
		t.Fatalf("posted count = %d, want 1", len(reporter.posted))
	}
	if reporter.posted[0].Disk.Name != "sdb" {
		t.Fatalf("posted disk = %+v, want sdb", reporter.posted[0].Disk)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("output file was not written: %v", err)
	}
	if len(data) == 0 || data[0] != '{' {
		t.Fatalf("output file does not contain JSON object: %q", string(data))
	}
}
