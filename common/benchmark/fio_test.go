package benchmark

import (
	"strings"
	"testing"
)

func TestRenderFIOJobIncludesProfileCasesAndSafetyDefaults(t *testing.T) {
	profile, err := SelectProfile("short")
	if err != nil {
		t.Fatalf("SelectProfile(short) returned error: %v", err)
	}

	job, err := RenderFIOJob(profile, DiskTarget{
		Name:          "nvme0n1",
		DevicePath:    "/dev/nvme0n1",
		MediaType:     "SSD",
		InterfaceType: "NVME",
	})
	if err != nil {
		t.Fatalf("RenderFIOJob returned error: %v", err)
	}

	required := []string{
		"[global]",
		"ioengine=libaio",
		"direct=1",
		"time_based=1",
		"runtime=60",
		"ramp_time=10",
		"output-format=json",
		"percentile_list=50:90:95:99:99.9",
		"[latency_probe_randread_4k]",
		"filename=/dev/nvme0n1",
		"rw=randread",
		"bs=4k",
		"numjobs=1",
		"iodepth=1",
		"[mixed_random_randrw_16k]",
		"rwmixread=70",
		"numjobs=4",
		"iodepth=32",
		"[sequential_perf_read_1M]",
		"offset=40%",
		"size=20%",
		"stonewall",
	}
	for _, token := range required {
		if !strings.Contains(job, token) {
			t.Fatalf("job file missing %q:\n%s", token, job)
		}
	}
}

func TestValidateFIOCasesRejectsMissingDuplicateAndUnknownCases(t *testing.T) {
	profile, err := SelectProfile("short")
	if err != nil {
		t.Fatalf("SelectProfile(short) returned error: %v", err)
	}

	complete := make([]FIOCaseResult, 0, len(profile.Cases))
	for _, c := range profile.Cases {
		complete = append(complete, FIOCaseResult{CaseID: c.ID, CaseOrder: c.Order})
	}
	if err := ValidateFIOCases(profile, complete); err != nil {
		t.Fatalf("ValidateFIOCases complete profile returned error: %v", err)
	}

	missing := complete[:len(complete)-1]
	if err := ValidateFIOCases(profile, missing); err == nil {
		t.Fatal("ValidateFIOCases should reject missing cases")
	}

	duplicate := append([]FIOCaseResult{}, complete...)
	duplicate[1].CaseID = duplicate[0].CaseID
	if err := ValidateFIOCases(profile, duplicate); err == nil {
		t.Fatal("ValidateFIOCases should reject duplicate cases")
	}

	unknown := append([]FIOCaseResult{}, complete...)
	unknown[0].CaseID = "unknown_case"
	if err := ValidateFIOCases(profile, unknown); err == nil {
		t.Fatal("ValidateFIOCases should reject unknown cases")
	}
}
