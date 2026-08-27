package block

import (
	"fmt"
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

func TestResolveMegacliDeviceNameBySerialDir(t *testing.T) {
	root := t.TempDir()
	devDir := filepath.Join(root, "dev")
	byIDDir := filepath.Join(root, "by-id")

	if err := os.Mkdir(devDir, 0o755); err != nil {
		t.Fatalf("mkdir dev dir: %v", err)
	}
	if err := os.Mkdir(byIDDir, 0o755); err != nil {
		t.Fatalf("mkdir by-id dir: %v", err)
	}

	devices := []string{"sda", "sdc", "sdd"}
	for i := 1; i <= 10; i++ {
		devices = append(devices, fmt.Sprintf("sda%d", i))
	}
	for _, device := range devices {
		if err := os.WriteFile(filepath.Join(devDir, device), []byte(device), 0o644); err != nil {
			t.Fatalf("write device %s: %v", device, err)
		}
	}

	// Mirrors a real JBOD system disk: 10 partitions, exposed under both the
	// ata-* and wwn-* naming schemes (22 symlinks for one physical disk).
	links := map[string]string{
		"ata-SAMSUNG_MZ7LH240HAHQ-00005_S45RNC0R606426": "sda",
		"wwn-0x5002538e7161ef05":                        "sda",
	}
	for i := 1; i <= 10; i++ {
		links[fmt.Sprintf("ata-SAMSUNG_MZ7LH240HAHQ-00005_S45RNC0R606426-part%d", i)] = fmt.Sprintf("sda%d", i)
		links[fmt.Sprintf("wwn-0x5002538e7161ef05-part%d", i)] = fmt.Sprintf("sda%d", i)
	}
	links["ata-SAMSUNG_MZ7LH960HAJR-00005_S45NNA0MA78131"] = "sdc"
	links["wwn-0x5002538e19a4c64a"] = "sdc"
	links["ata-HGST_HUS726T6TALE6L4_V8JZDMKR"] = "sdd"

	for linkName, device := range links {
		if err := os.Symlink(filepath.Join(devDir, device), filepath.Join(byIDDir, linkName)); err != nil {
			t.Fatalf("symlink %s: %v", linkName, err)
		}
	}

	t.Run("partitioned disk resolves to the whole disk only", func(t *testing.T) {
		got := resolveMegacliDeviceNameBySerialDir(byIDDir, "S45RNC0R606426")
		if got != "sda" {
			t.Fatalf("device = %q, want %q", got, "sda")
		}
	})

	t.Run("unpartitioned disk resolves", func(t *testing.T) {
		got := resolveMegacliDeviceNameBySerialDir(byIDDir, "S45NNA0MA78131")
		if got != "sdc" {
			t.Fatalf("device = %q, want %q", got, "sdc")
		}
	})

	t.Run("single naming scheme resolves", func(t *testing.T) {
		got := resolveMegacliDeviceNameBySerialDir(byIDDir, "V8JZDMKR")
		if got != "sdd" {
			t.Fatalf("device = %q, want %q", got, "sdd")
		}
	})

	t.Run("empty serial returns empty", func(t *testing.T) {
		if got := resolveMegacliDeviceNameBySerialDir(byIDDir, ""); got != "" {
			t.Fatalf("device = %q, want empty string", got)
		}
		if got := resolveMegacliDeviceNameBySerialDir(byIDDir, "   "); got != "" {
			t.Fatalf("device = %q, want empty string", got)
		}
	})

	t.Run("unknown serial returns empty", func(t *testing.T) {
		if got := resolveMegacliDeviceNameBySerialDir(byIDDir, "NOSUCHSERIAL"); got != "" {
			t.Fatalf("device = %q, want empty string", got)
		}
	})

	t.Run("serial with regex metacharacters is matched literally", func(t *testing.T) {
		if got := resolveMegacliDeviceNameBySerialDir(byIDDir, "S45RNC0R60642."); got != "" {
			t.Fatalf("device = %q, want empty string", got)
		}
		if got := resolveMegacliDeviceNameBySerialDir(byIDDir, ".*"); got != "" {
			t.Fatalf("device = %q, want empty string", got)
		}
	})

	t.Run("ambiguous serial returns empty", func(t *testing.T) {
		if err := os.Symlink(filepath.Join(devDir, "sdd"), filepath.Join(byIDDir, "scsi-SHARED_SERIAL_V8JZDMKR")); err != nil {
			t.Fatalf("add first ambiguous symlink: %v", err)
		}
		if err := os.Symlink(filepath.Join(devDir, "sdc"), filepath.Join(byIDDir, "scsi-OTHER_V8JZDMKR")); err != nil {
			t.Fatalf("add second ambiguous symlink: %v", err)
		}
		if got := resolveMegacliDeviceNameBySerialDir(byIDDir, "V8JZDMKR"); got != "" {
			t.Fatalf("device = %q, want empty string", got)
		}
	})

	t.Run("missing directory returns empty", func(t *testing.T) {
		got := resolveMegacliDeviceNameBySerialDir(filepath.Join(root, "absent"), "S45RNC0R606426")
		if got != "" {
			t.Fatalf("device = %q, want empty string", got)
		}
	})
}
