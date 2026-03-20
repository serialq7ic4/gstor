package cmd

import (
	"errors"
	"testing"

	"github.com/chenq7an/gstor/common/block"
)

func TestResolveSlotSmartReadTargetWithResolvers(t *testing.T) {
	t.Run("raid slot does not fall back to mapped device", func(t *testing.T) {
		slot := block.SlotID{ControllerID: "0", EnclosureID: "251", SlotID: "4"}
		mappedCalled := false

		_, err := resolveSlotSmartReadTargetWithResolvers(
			slot,
			func(got block.SlotID) (smartReadTarget, error) {
				if got != slot {
					t.Fatalf("raid resolver got slot %+v, want %+v", got, slot)
				}
				return smartReadTarget{}, errors.New("raid smart failed")
			},
			func(block.SlotID) (smartReadTarget, error) {
				mappedCalled = true
				return smartReadTarget{DeviceLabel: "sdb"}, nil
			},
		)
		if err == nil {
			t.Fatal("expected raid resolver error")
		}
		if mappedCalled {
			t.Fatal("mapped resolver should not be called for c:e:s slots")
		}
	})

	t.Run("controller slot uses mapped device", func(t *testing.T) {
		slot := block.SlotID{ControllerID: "0", SlotID: "7"}

		target, err := resolveSlotSmartReadTargetWithResolvers(
			slot,
			func(block.SlotID) (smartReadTarget, error) {
				t.Fatal("raid resolver should not be called for c:s slots")
				return smartReadTarget{}, nil
			},
			func(got block.SlotID) (smartReadTarget, error) {
				if got != slot {
					t.Fatalf("mapped resolver got slot %+v, want %+v", got, slot)
				}
				return smartReadTarget{DeviceLabel: "sdc"}, nil
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if target.DeviceLabel != "sdc" {
			t.Fatalf("resolved device = %q, want %q", target.DeviceLabel, "sdc")
		}
	})
}

func TestResolveNamedSmartReadTarget(t *testing.T) {
	t.Run("unique raid-backed disk uses raid smart access", func(t *testing.T) {
		devices := []block.Disk{
			{Name: "sdb", CES: "0:251:4"},
		}

		target, matched, err := resolveNamedSmartReadTarget(
			"sdb",
			devices,
			func(slot block.SlotID) (smartReadTarget, error) {
				if slot.String() != "0:251:4" {
					t.Fatalf("raid resolver got slot %s, want 0:251:4", slot.String())
				}
				return smartReadTarget{
					DeviceLabel: "megaraid:6@/dev/bus/0",
					Slot:        slot.String(),
					ReadCommand: "smartctl ...",
				}, nil
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !matched {
			t.Fatal("expected device name to match collected disks")
		}
		if target.DeviceLabel != "megaraid:6@/dev/bus/0" {
			t.Fatalf("resolved device = %q", target.DeviceLabel)
		}
		if target.Slot != "0:251:4" {
			t.Fatalf("resolved slot = %q", target.Slot)
		}
	})

	t.Run("ambiguous raid-backed disk returns error", func(t *testing.T) {
		devices := []block.Disk{
			{Name: "sda", CES: "0:251:2"},
			{Name: "sda", CES: "0:251:3"},
		}

		_, matched, err := resolveNamedSmartReadTarget("sda", devices, func(block.SlotID) (smartReadTarget, error) {
			t.Fatal("raid resolver should not be called for ambiguous disks")
			return smartReadTarget{}, nil
		})
		if !matched {
			t.Fatal("expected device name to match collected disks")
		}
		if err == nil {
			t.Fatal("expected ambiguous disk error")
		}
		want := "device sda maps to multiple slots (0:251:2, 0:251:3); use c:e:s to read SMART"
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("direct disk keeps direct smart target", func(t *testing.T) {
		devices := []block.Disk{
			{Name: "nvme0n1", CES: "22"},
		}

		target, matched, err := resolveNamedSmartReadTarget("nvme0n1", devices, func(block.SlotID) (smartReadTarget, error) {
			t.Fatal("raid resolver should not be called for direct disks")
			return smartReadTarget{}, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !matched {
			t.Fatal("expected device name to match collected disks")
		}
		if target.DeviceLabel != "nvme0n1" {
			t.Fatalf("resolved device = %q, want %q", target.DeviceLabel, "nvme0n1")
		}
		if target.Slot != "22" {
			t.Fatalf("resolved slot = %q, want %q", target.Slot, "22")
		}
		if target.ReadCommand != "smartctl -i -H -A /dev/nvme0n1" {
			t.Fatalf("read command = %q", target.ReadCommand)
		}
	})
}

func TestPCIBusCandidates(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "storcli decimal bus number also tries hex pci path",
			input: "49",
			want:  []string{"49", "31"},
		},
		{
			name:  "hex bus number keeps hex path",
			input: "0x31",
			want:  []string{"31"},
		},
		{
			name:  "single digit bus number is zero padded once",
			input: "7",
			want:  []string{"07"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pciBusCandidates(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("candidate count = %d, want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("candidate[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
