package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/chenq7an/gstor/common/utils"
)

const defaultLockPath = "/var/lock/gstor-benchmark.lock"

type ServerInfo struct {
	Hostname  string `json:"hostname"`
	SN        string `json:"sn,omitempty"`
	PrimaryIP string `json:"primary_ip,omitempty"`
}

type ProfileSummary struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	Hash              string `json:"hash"`
	BaselineCandidate bool   `json:"baseline_candidate"`
}

type BenchmarkWindow struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

type DiskBenchmarkResult struct {
	Server    ServerInfo      `json:"server"`
	Disk      DiskTarget      `json:"disk"`
	Profile   ProfileSummary  `json:"profile"`
	Benchmark BenchmarkWindow `json:"benchmark"`
	FIOCases  []FIOCaseResult `json:"fio_cases"`
}

type RunOutput struct {
	Profile ProfileSummary        `json:"profile"`
	Results []DiskBenchmarkResult `json:"results"`
}

type DiskDiscoverer func(ctx context.Context, explicit []string) ([]DiskTarget, []SkippedDisk, error)

type FIORunner interface {
	RunFIO(ctx context.Context, profile Profile, disk DiskTarget) ([]FIOCaseResult, error)
}

type Reporter interface {
	Post(ctx context.Context, url string, result DiskBenchmarkResult) error
}

type ServerInfoProvider func(ctx context.Context) (ServerInfo, error)

type RunOptions struct {
	ProfileName        string
	Disks              []string
	OutputPath         string
	Format             string
	ReportURL          string
	LockPath           string
	Probe              Probe
	Discoverer         DiskDiscoverer
	FIORunner          FIORunner
	Reporter           Reporter
	ServerInfoProvider ServerInfoProvider
	Stderr             io.Writer
}

func RunBenchmark(ctx context.Context, options RunOptions) (RunOutput, error) {
	if options.Format != "" && options.Format != "json" {
		return RunOutput{}, fmt.Errorf("unsupported benchmark output format %q", options.Format)
	}

	profile, err := SelectProfile(options.ProfileName)
	if err != nil {
		return RunOutput{}, err
	}

	probe := options.Probe
	if probe == nil {
		probe = SystemProbe{}
	}
	requirements := CheckRequirements(probe)
	if err := requirements.Error(); err != nil {
		return RunOutput{}, err
	}

	unlock, err := acquireLock(lockPath(options.LockPath))
	if err != nil {
		return RunOutput{}, err
	}
	defer unlock()

	discoverer := options.Discoverer
	if discoverer == nil {
		inspector := NewSystemInspector()
		discoverer = func(ctx context.Context, explicit []string) ([]DiskTarget, []SkippedDisk, error) {
			return DiscoverEligibleDisks(ctx, inspector, explicit)
		}
	}

	disks, skipped, err := discoverer(ctx, options.Disks)
	if err != nil {
		return RunOutput{}, err
	}
	for _, skippedDisk := range skipped {
		writeProgress(options.Stderr, "skip %s: %s\n", skippedDisk.Name, skippedDisk.Reason)
	}
	if len(disks) == 0 {
		return RunOutput{}, fmt.Errorf("no eligible bare disks found")
	}
	writeProgress(options.Stderr, "benchmark plan: profile=%s_%s disks=%d\n", profile.Name, profile.Version, len(disks))

	fioRunner := options.FIORunner
	if fioRunner == nil {
		fioRunner = RealFIORunner{Runner: SystemRunner{}}
	}
	reporter := options.Reporter
	if reporter == nil {
		reporter = HTTPReporter{Client: &http.Client{Timeout: 30 * time.Second}}
	}
	serverInfoProvider := options.ServerInfoProvider
	if serverInfoProvider == nil {
		serverInfoProvider = DefaultServerInfoProvider(SystemRunner{})
	}
	server, err := serverInfoProvider(ctx)
	if err != nil {
		return RunOutput{}, err
	}

	output := RunOutput{Profile: profileSummary(profile)}
	for _, disk := range disks {
		writeProgress(options.Stderr, "running benchmark disk=%s path=%s\n", disk.Name, disk.DevicePath)
		started := time.Now().UTC()
		cases, err := fioRunner.RunFIO(ctx, profile, disk)
		if err != nil {
			return output, err
		}
		if err := ValidateFIOCases(profile, cases); err != nil {
			return output, err
		}
		result := DiskBenchmarkResult{
			Server:    server,
			Disk:      disk,
			Profile:   profileSummary(profile),
			Benchmark: BenchmarkWindow{StartedAt: started, FinishedAt: time.Now().UTC()},
			FIOCases:  cases,
		}
		if options.ReportURL != "" {
			if err := reporter.Post(ctx, options.ReportURL, result); err != nil {
				return output, err
			}
		}
		output.Results = append(output.Results, result)
	}

	if options.OutputPath != "" {
		if err := writeJSONFile(options.OutputPath, output); err != nil {
			return output, err
		}
	}

	return output, nil
}

func writeProgress(writer io.Writer, format string, args ...any) {
	if writer == nil {
		return
	}
	_, _ = fmt.Fprintf(writer, format, args...)
}

func profileSummary(profile Profile) ProfileSummary {
	return ProfileSummary{
		Name:              profile.Name,
		Version:           profile.Version,
		Hash:              profile.Hash,
		BaselineCandidate: profile.BaselineCandidate,
	}
}

func lockPath(path string) string {
	if path == "" {
		return defaultLockPath
	}
	return path
}

func acquireLock(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open benchmark lock %s: %w", path, err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("another benchmark run is active: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}

func writeJSONFile(path string, output RunOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark output: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write benchmark output %s: %w", path, err)
	}
	return nil
}

type RealFIORunner struct {
	Runner  CommandRunner
	TempDir string
}

func (r RealFIORunner) RunFIO(ctx context.Context, profile Profile, disk DiskTarget) ([]FIOCaseResult, error) {
	runner := r.Runner
	if runner == nil {
		runner = SystemRunner{}
	}
	job, err := RenderFIOJob(profile, disk)
	if err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(r.TempDir, "gstor-fio-*.fio")
	if err != nil {
		return nil, fmt.Errorf("failed to create fio job file: %w", err)
	}
	jobPath := tmp.Name()
	defer func() { _ = os.Remove(jobPath) }()
	if _, err := tmp.WriteString(job); err != nil {
		_ = tmp.Close()
		return nil, fmt.Errorf("failed to write fio job file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("failed to close fio job file: %w", err)
	}

	result, err := runner.Run(ctx, "fio", "--output-format=json", jobPath)
	if err != nil {
		return nil, fmt.Errorf("fio failed for %s: %w: %s", disk.DevicePath, err, result.Stderr)
	}
	return ParseFIOOutput(profile, []byte(result.Stdout))
}

func ParseFIOOutput(profile Profile, data []byte) ([]FIOCaseResult, error) {
	var document struct {
		Jobs []map[string]any `json:"jobs"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("failed to parse fio JSON: %w", err)
	}
	definitions := make(map[string]CaseDefinition, len(profile.Cases))
	for _, c := range profile.Cases {
		definitions[c.ID] = c
	}
	cases := make([]FIOCaseResult, 0, len(document.Jobs))
	for _, job := range document.Jobs {
		name, _ := job["jobname"].(string)
		definition, ok := definitions[name]
		if !ok {
			cases = append(cases, FIOCaseResult{CaseID: name})
			continue
		}
		metrics := make(map[string]any)
		if read, ok := job["read"]; ok {
			metrics["read"] = read
		}
		if write, ok := job["write"]; ok {
			metrics["write"] = write
		}
		cases = append(cases, FIOCaseResult{
			CaseID:    definition.ID,
			CaseOrder: definition.Order,
			Parameters: map[string]any{
				"rw":        definition.RW,
				"bs":        definition.BlockSize,
				"rwmixread": definition.RWMixRead,
				"offset":    definition.Offset,
				"size":      definition.Size,
			},
			Metrics:    metrics,
			RawFIOJSON: job,
		})
	}
	return cases, nil
}

type HTTPReporter struct {
	Client *http.Client
}

func (r HTTPReporter) Post(ctx context.Context, url string, result DiskBenchmarkResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark report: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create benchmark report request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to post benchmark result: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("benchmark report API returned %s: %s", response.Status, string(body))
	}
	return nil
}

func DefaultServerInfoProvider(runner CommandRunner) ServerInfoProvider {
	return func(ctx context.Context) (ServerInfo, error) {
		hostname, err := os.Hostname()
		if err != nil {
			return ServerInfo{}, fmt.Errorf("failed to get hostname: %w", err)
		}
		primaryIP, _ := utils.PrimaryIPv4()
		result, _ := runner.Run(ctx, "dmidecode", "-s", "system-serial-number")
		return ServerInfo{
			Hostname:  hostname,
			SN:        result.Stdout,
			PrimaryIP: primaryIP,
		}, nil
	}
}

func ResultArchiveName(version string, goos string, goarch string) string {
	return filepath.Base(fmt.Sprintf("gstor-%s-%s-%s.tar.gz", version, goos, goarch))
}
