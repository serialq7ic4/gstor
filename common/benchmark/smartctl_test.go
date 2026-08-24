package benchmark

import (
	"context"
	"strings"
	"testing"
)

func TestCheckNVMeSmartctlCompatibilityRequiresMinimumVersion(t *testing.T) {
	tests := []struct {
		name        string
		smartctlOut string
		wantReady   bool
	}{
		{name: "CentOS 7 stock version", smartctlOut: "smartctl 6.6 2016-05-31", wantReady: false},
		{name: "older than minimum", smartctlOut: "smartctl 7.0.1 2018-10-02", wantReady: false},
		{name: "minimum version", smartctlOut: "smartctl 7.0.2 2018-12-30", wantReady: true},
		{name: "newer version", smartctlOut: "smartctl 7.3 2022-02-28", wantReady: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeCommandRunner{
				results: []fakeCommandResult{
					{result: CommandResult{Stdout: "nvme0n1 disk\n"}},
					{result: CommandResult{Stdout: tt.smartctlOut}},
				},
			}

			check, err := CheckNVMeSmartctlCompatibility(context.Background(), runner)
			if err != nil {
				t.Fatalf("CheckNVMeSmartctlCompatibility returned error: %v", err)
			}
			if check.Ready != tt.wantReady {
				t.Fatalf("check.Ready = %t, want %t: %+v", check.Ready, tt.wantReady, check)
			}
			if !check.HasNVMe {
				t.Fatalf("check.HasNVMe = false, want true: %+v", check)
			}
		})
	}
}

func TestCheckNVMeSmartctlCompatibilitySkipsVersionCheckWithoutNVMe(t *testing.T) {
	runner := &fakeCommandRunner{
		results: []fakeCommandResult{{result: CommandResult{Stdout: "sdb disk\n"}}},
	}

	check, err := CheckNVMeSmartctlCompatibility(context.Background(), runner)
	if err != nil {
		t.Fatalf("CheckNVMeSmartctlCompatibility returned error: %v", err)
	}
	if check.HasNVMe || !check.Ready {
		t.Fatalf("check = %+v, want no NVMe and ready", check)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %#v, want only lsblk", runner.calls)
	}
}

func TestFilterTargetsForNVMeSmartctlCompatibilitySkipsOnlyNVMe(t *testing.T) {
	check := NVMeSmartctlCheck{
		HasNVMe: true,
		Ready:   false,
		Version: "7.0.1",
	}
	targets := []DiskTarget{
		{Name: "sdb", DevicePath: "/dev/sdb", MediaType: "HDD", InterfaceType: "SATA"},
		{Name: "nvme0n1", DevicePath: "/dev/nvme0n1", MediaType: "SSD", InterfaceType: "NVME"},
	}

	filtered, skipped, err := filterTargetsForNVMeSmartctl(targets, check, nil)
	if err != nil {
		t.Fatalf("filterTargetsForNVMeSmartctl returned error: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Name != "sdb" {
		t.Fatalf("filtered = %+v, want only sdb", filtered)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0].Reason, "smartmontools") {
		t.Fatalf("skipped = %+v, want smartmontools reason", skipped)
	}
}

func TestFilterTargetsForNVMeSmartctlCompatibilityRejectsExplicitNVMe(t *testing.T) {
	check := NVMeSmartctlCheck{HasNVMe: true, Ready: false, Version: "6.6"}
	targets := []DiskTarget{{Name: "nvme0n1", DevicePath: "/dev/nvme0n1", MediaType: "SSD", InterfaceType: "NVME"}}

	_, _, err := filterTargetsForNVMeSmartctl(targets, check, []string{"nvme0n1"})
	if err == nil || !strings.Contains(err.Error(), "smartmontools") {
		t.Fatalf("error = %v, want smartmontools error", err)
	}
}
