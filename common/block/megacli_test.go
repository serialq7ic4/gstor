package block

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMegacliDeviceNameByPathDir(t *testing.T) {
	root := t.TempDir()
	devDir := filepath.Join(root, "dev")
	byPathDir := filepath.Join(root, "by-path")

	if err := os.Mkdir(devDir, 0o755); err != nil {
		t.Fatalf("mkdir dev dir: %v", err)
	}
	if err := os.Mkdir(byPathDir, 0o755); err != nil {
		t.Fatalf("mkdir by-path dir: %v", err)
	}

	for _, device := range []string{"sda", "sdb", "sdc"} {
		if err := os.WriteFile(filepath.Join(devDir, device), []byte(device), 0o644); err != nil {
			t.Fatalf("write device %s: %v", device, err)
		}
	}

	links := map[string]string{
		"pci-0000:01:00.0-scsi-0:2:0:0":       "sda",
		"pci-0000:01:00.0-scsi-0:2:0:0-part1": "sda",
		"pci-0000:01:00.0-scsi-0:2:1:0":       "sdb",
		"pci-0000:01:00.0-scsi-0:3:2:0":       "sdc",
	}
	for linkName, device := range links {
		if err := os.Symlink(filepath.Join(devDir, device), filepath.Join(byPathDir, linkName)); err != nil {
			t.Fatalf("symlink %s: %v", linkName, err)
		}
	}

	t.Run("exact sequence match wins", func(t *testing.T) {
		got := resolveMegacliDeviceNameByPathDir(byPathDir, "01", "2", "1")
		if got != "sdb" {
			t.Fatalf("device = %q, want %q", got, "sdb")
		}
	})

	t.Run("fallback matches target id when sequence number is wrong", func(t *testing.T) {
		got := resolveMegacliDeviceNameByPathDir(byPathDir, "01", "4", "0")
		if got != "sda" {
			t.Fatalf("device = %q, want %q", got, "sda")
		}
	})

	t.Run("ambiguous target id returns empty", func(t *testing.T) {
		if err := os.Symlink(filepath.Join(devDir, "sdc"), filepath.Join(byPathDir, "pci-0000:01:00.0-scsi-0:3:0:0")); err != nil {
			t.Fatalf("add ambiguous symlink: %v", err)
		}
		got := resolveMegacliDeviceNameByPathDir(byPathDir, "01", "9", "0")
		if got != "" {
			t.Fatalf("device = %q, want empty string", got)
		}
	})
}
