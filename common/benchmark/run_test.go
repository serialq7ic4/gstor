package benchmark

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
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

type sequencedFIORunner struct {
	responses []fioResponse
}

type fioResponse struct {
	cases []FIOCaseResult
	err   error
}

func (r *sequencedFIORunner) RunFIO(ctx context.Context, profile Profile, disk DiskTarget) ([]FIOCaseResult, error) {
	if len(r.responses) == 0 {
		return nil, errors.New("unexpected fio call")
	}
	response := r.responses[0]
	r.responses = r.responses[1:]
	return response.cases, response.err
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

func TestRunBenchmarkPrintsEstimatedDuration(t *testing.T) {
	profile, err := SelectProfile("short")
	if err != nil {
		t.Fatalf("SelectProfile(short): %v", err)
	}
	stderr := &bytes.Buffer{}
	fio := &fakeFIORunner{cases: completeCaseResults(t, profile)}

	_, err = RunBenchmark(context.Background(), RunOptions{
		ProfileName: "short",
		Probe: fakeProbe{
			paths: map[string]string{
				"fio": "/usr/bin/fio", "lsblk": "/usr/bin/lsblk", "blkid": "/usr/sbin/blkid",
				"findmnt": "/usr/bin/findmnt", "fuser": "/usr/bin/fuser", "smartctl": "/usr/sbin/smartctl",
			},
			euid: 0,
		},
		Discoverer: func(context.Context, []string) ([]DiskTarget, []SkippedDisk, error) {
			return []DiskTarget{
				{Name: "sdb", DevicePath: "/dev/sdb", MediaType: "HDD", InterfaceType: "SATA"},
				{Name: "sdc", DevicePath: "/dev/sdc", MediaType: "HDD", InterfaceType: "SATA"},
			}, nil, nil
		},
		FIORunner: fio,
		ServerInfoProvider: func(context.Context) (ServerInfo, error) {
			return ServerInfo{Hostname: "host-a"}, nil
		},
		LockPath: filepath.Join(t.TempDir(), "gstor.lock"),
		Stderr:   stderr,
	})
	if err != nil {
		t.Fatalf("RunBenchmark returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "estimated_duration=16m20s") {
		t.Fatalf("stderr = %q, want estimated_duration=16m20s", stderr.String())
	}
}

func TestRunBenchmarkSkipsNVMeWhenSmartctlIsTooOld(t *testing.T) {
	profile, err := SelectProfile("short")
	if err != nil {
		t.Fatalf("SelectProfile(short): %v", err)
	}
	fio := &fakeFIORunner{cases: completeCaseResults(t, profile)}
	stderr := &bytes.Buffer{}

	output, err := RunBenchmark(context.Background(), RunOptions{
		ProfileName: "short",
		Probe: fakeProbe{
			paths: map[string]string{
				"fio": "/usr/bin/fio", "lsblk": "/usr/bin/lsblk", "blkid": "/usr/sbin/blkid",
				"findmnt": "/usr/bin/findmnt", "fuser": "/usr/bin/fuser", "smartctl": "/usr/sbin/smartctl",
			},
			euid: 0,
		},
		Discoverer: func(context.Context, []string) ([]DiskTarget, []SkippedDisk, error) {
			return []DiskTarget{
				{Name: "sdb", DevicePath: "/dev/sdb", MediaType: "HDD", InterfaceType: "SATA"},
				{Name: "nvme0n1", DevicePath: "/dev/nvme0n1", MediaType: "SSD", InterfaceType: "NVME"},
			}, nil, nil
		},
		SmartctlVersionChecker: func(context.Context) (SmartctlVersionStatus, error) {
			return SmartctlVersionStatus{Version: "7.0.1", Ready: false, Warning: "NVMe disks require smartmontools >= 7.0.2, detected 7.0.1"}, nil
		},
		FIORunner: fio,
		ServerInfoProvider: func(context.Context) (ServerInfo, error) {
			return ServerInfo{Hostname: "host-a"}, nil
		},
		LockPath: filepath.Join(t.TempDir(), "gstor.lock"),
		Stderr:   stderr,
	})
	if err != nil {
		t.Fatalf("RunBenchmark returned error: %v", err)
	}
	if len(output.Results) != 1 || output.Results[0].Disk.Name != "sdb" {
		t.Fatalf("output = %+v, want only sdb", output)
	}
	if fio.calls != 1 {
		t.Fatalf("fio calls = %d, want 1 for sdb only", fio.calls)
	}
	if !strings.Contains(stderr.String(), "skip nvme0n1") || !strings.Contains(stderr.String(), "smartmontools") {
		t.Fatalf("stderr = %q, want NVMe smartmontools skip message", stderr.String())
	}
}

func TestRunBenchmarkRetainsCompletedDiskOutputWhenLaterDiskFails(t *testing.T) {
	profile, err := SelectProfile("short")
	if err != nil {
		t.Fatalf("SelectProfile(short): %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "benchmark.json")
	fio := &sequencedFIORunner{responses: []fioResponse{
		{cases: completeCaseResults(t, profile)},
		{err: errors.New("fio interrupted")},
	}}

	output, err := RunBenchmark(context.Background(), RunOptions{
		ProfileName: "short",
		OutputPath:  outputPath,
		Probe: fakeProbe{
			paths: map[string]string{
				"fio": "/usr/bin/fio", "lsblk": "/usr/bin/lsblk", "blkid": "/usr/sbin/blkid",
				"findmnt": "/usr/bin/findmnt", "fuser": "/usr/bin/fuser", "smartctl": "/usr/sbin/smartctl",
			},
			euid: 0,
		},
		Discoverer: func(context.Context, []string) ([]DiskTarget, []SkippedDisk, error) {
			return []DiskTarget{
				{Name: "sdb", DevicePath: "/dev/sdb", MediaType: "HDD", InterfaceType: "SATA"},
				{Name: "sdc", DevicePath: "/dev/sdc", MediaType: "HDD", InterfaceType: "SATA"},
			}, nil, nil
		},
		FIORunner: fio,
		ServerInfoProvider: func(context.Context) (ServerInfo, error) {
			return ServerInfo{Hostname: "host-a"}, nil
		},
		LockPath: filepath.Join(t.TempDir(), "gstor.lock"),
	})
	if err == nil {
		t.Fatal("RunBenchmark should return the later disk error")
	}
	if len(output.Results) != 1 || output.Results[0].Disk.Name != "sdb" {
		t.Fatalf("partial output = %+v, want completed sdb only", output)
	}
	data, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("partial output file was not written: %v", readErr)
	}
	if !strings.Contains(string(data), `"name": "sdb"`) || strings.Contains(string(data), `"name": "sdc"`) {
		t.Fatalf("partial output file = %s, want only completed disk", string(data))
	}
}

func TestSystemRunnerCancelsProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group signaling is Unix-specific")
	}

	marker := filepath.Join(t.TempDir(), "leaked-child")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	script := "trap 'exit 0' TERM; (sleep 1; echo leaked > " + marker + ") & wait"

	_, _ = SystemRunner{}.Run(ctx, "sh", "-c", script)
	time.Sleep(1500 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("child process survived cancellation and wrote marker")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat marker: %v", err)
	}
}

func TestRealFIORunnerPassesOutputFormatOnCommandLine(t *testing.T) {
	profile, err := SelectProfile("short")
	if err != nil {
		t.Fatalf("SelectProfile(short): %v", err)
	}
	runner := &fakeCommandRunner{
		results: []fakeCommandResult{{
			result: CommandResult{Stdout: `{"jobs":[]}`},
		}},
	}

	_, err = RealFIORunner{Runner: runner, TempDir: t.TempDir()}.RunFIO(context.Background(), profile, DiskTarget{
		Name:          "sdb",
		DevicePath:    "/dev/sdb",
		MediaType:     "HDD",
		InterfaceType: "UNKNOWN",
	})
	if err != nil {
		t.Fatalf("RunFIO returned error: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %#v, want one fio call", runner.calls)
	}
	call := runner.calls[0]
	if len(call) != 3 {
		t.Fatalf("fio call = %#v, want fio --output-format=json <jobfile>", call)
	}
	if !reflect.DeepEqual(call[:2], []string{"fio", "--output-format=json"}) {
		t.Fatalf("fio call prefix = %#v, want fio --output-format=json", call[:2])
	}
}
