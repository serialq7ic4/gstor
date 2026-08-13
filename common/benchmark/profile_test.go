package benchmark

import "testing"

func TestSelectProfileDefault(t *testing.T) {
	profile, err := SelectProfile("default")
	if err != nil {
		t.Fatalf("SelectProfile(default) returned error: %v", err)
	}

	if profile.Name != "default" {
		t.Fatalf("profile name = %q, want default", profile.Name)
	}
	if profile.Version != "v1" {
		t.Fatalf("profile version = %q, want v1", profile.Version)
	}
	if !profile.BaselineCandidate {
		t.Fatal("default profile should be baseline candidate")
	}
	if profile.Hash == "" {
		t.Fatal("profile hash must be populated")
	}
	if len(profile.Cases) != 25 {
		t.Fatalf("default profile case count = %d, want 25", len(profile.Cases))
	}

	first := profile.Cases[0]
	if first.ID != "latency_probe_randread_4k" || first.Order != 1 || first.RW != "randread" || first.BlockSize != "4k" {
		t.Fatalf("unexpected first case: %+v", first)
	}

	last := profile.Cases[len(profile.Cases)-1]
	if last.ID != "sequential_perf_write_4M" || last.Order != 25 || last.RW != "write" || last.BlockSize != "4M" {
		t.Fatalf("unexpected last case: %+v", last)
	}
}

func TestSelectProfileShort(t *testing.T) {
	profile, err := SelectProfile("short")
	if err != nil {
		t.Fatalf("SelectProfile(short) returned error: %v", err)
	}

	if profile.Name != "short" {
		t.Fatalf("profile name = %q, want short", profile.Name)
	}
	if profile.Version != "v1" {
		t.Fatalf("profile version = %q, want v1", profile.Version)
	}
	if profile.BaselineCandidate {
		t.Fatal("short profile should not be baseline candidate")
	}
	if len(profile.Cases) != 7 {
		t.Fatalf("short profile case count = %d, want 7", len(profile.Cases))
	}

	wantIDs := []string{
		"latency_probe_randread_4k",
		"latency_probe_randwrite_4k",
		"random_perf_randread_4k",
		"random_perf_randwrite_4k",
		"mixed_random_randrw_16k",
		"sequential_perf_read_1M",
		"sequential_perf_write_1M",
	}
	for i, want := range wantIDs {
		if profile.Cases[i].ID != want {
			t.Fatalf("short case %d = %q, want %q", i, profile.Cases[i].ID, want)
		}
	}
}

func TestProfilePressureByMedia(t *testing.T) {
	tests := []struct {
		name        string
		target      DiskTarget
		phase       string
		wantNumJobs int
		wantIODepth int
	}{
		{
			name:        "sata hdd random",
			target:      DiskTarget{MediaType: "HDD", InterfaceType: "SATA"},
			phase:       "random_perf",
			wantNumJobs: 1,
			wantIODepth: 4,
		},
		{
			name:        "sata ssd sequential",
			target:      DiskTarget{MediaType: "SSD", InterfaceType: "SATA"},
			phase:       "sequential_perf",
			wantNumJobs: 1,
			wantIODepth: 16,
		},
		{
			name:        "nvme ssd mixed",
			target:      DiskTarget{MediaType: "SSD", InterfaceType: "NVME"},
			phase:       "mixed_random",
			wantNumJobs: 4,
			wantIODepth: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PressureFor(tt.target, tt.phase)
			if got.NumJobs != tt.wantNumJobs || got.IODepth != tt.wantIODepth {
				t.Fatalf("PressureFor(%+v, %s) = %+v, want numjobs=%d iodepth=%d", tt.target, tt.phase, got, tt.wantNumJobs, tt.wantIODepth)
			}
		})
	}
}
