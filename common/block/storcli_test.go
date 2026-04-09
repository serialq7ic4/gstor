package block

import (
	"reflect"
	"testing"
)

func TestParseStorcliDiscoveryOutput(t *testing.T) {
	t.Run("parses enclosure based slots", func(t *testing.T) {
		output := `
Drive Information :
=================

----------------------------------------------------------------------------
EID:Slt DID State DG       Size Intf Med SED PI SeSz Model               Sp
----------------------------------------------------------------------------
24:15    10 Onln  0  893.137 GB SATA SSD N   N  512B INTEL SSDSC2KB960G7 U
24:16    11 Onln  0  893.137 GB SATA SSD N   N  512B INTEL SSDSC2KB960G7 U
----------------------------------------------------------------------------
`

		got := parseStorcliDiscoveryOutput(output, 0)
		want := []string{"0:24:15", "0:24:16"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("device ids = %v, want %v", got, want)
		}
	})

	t.Run("parses slot only output when enclosure is missing", func(t *testing.T) {
		output := `
Drive Information :
=================

----------------------------------------------------------------------------
EID:Slt DID State DG       Size Intf Med SED PI SeSz Model               Sp
----------------------------------------------------------------------------
 :0       0 Onln  -  222.585 GB SATA SSD N   N  512B INTEL SSDSC2KB240G7 U
 :1       1 Onln  -  222.585 GB SATA SSD N   N  512B INTEL SSDSC2KB240G7 U
 :4       2 UGood -  893.137 GB SATA SSD N   N  512B INTEL SSDSC2KB960G7 U
----------------------------------------------------------------------------
`

		got := parseStorcliDiscoveryOutput(output, 0)
		want := []string{"0:0", "0:1", "0:4"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("device ids = %v, want %v", got, want)
		}
	})
}

func TestDiscoverStorcliDeviceIDsWithFallback(t *testing.T) {
	t.Run("prefers enclosure discovery when eid is available", func(t *testing.T) {
		eallOutput := `
Drive Information :
=================

----------------------------------------------------------------------------
EID:Slt DID State DG       Size Intf Med SED PI SeSz Model               Sp
----------------------------------------------------------------------------
24:15    10 Onln  0  893.137 GB SATA SSD N   N  512B INTEL SSDSC2KB960G7 U
24:16    11 Onln  0  893.137 GB SATA SSD N   N  512B INTEL SSDSC2KB960G7 U
----------------------------------------------------------------------------
`
		sallOutput := `
Drive Information :
=================

----------------------------------------------------------------------------
EID:Slt DID State DG       Size Intf Med SED PI SeSz Model               Sp
----------------------------------------------------------------------------
 :0       0 Onln  -  222.585 GB SATA SSD N   N  512B INTEL SSDSC2KB240G7 U
 :1       1 Onln  -  222.585 GB SATA SSD N   N  512B INTEL SSDSC2KB240G7 U
----------------------------------------------------------------------------
`

		got := discoverStorcliDeviceIDsFromOutputs(eallOutput, sallOutput, 0)
		want := []string{"0:24:15", "0:24:16"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("device ids = %v, want %v", got, want)
		}
	})

	t.Run("falls back to slot-only discovery when enclosure lookup has no disks", func(t *testing.T) {
		eallOutput := `
CLI Version = 007.0912.0000.0000 Dec 27, 2018
Operating system = Linux 3.10.0-693.el7.x86_64
Controller = 0
Status = Failure
Description = No Enclosure found.
`
		sallOutput := `
Drive Information :
=================

----------------------------------------------------------------------------
EID:Slt DID State DG       Size Intf Med SED PI SeSz Model               Sp
----------------------------------------------------------------------------
 :0       0 Onln  -  222.585 GB SATA SSD N   N  512B INTEL SSDSC2KB240G7 U
 :1       1 Onln  -  222.585 GB SATA SSD N   N  512B INTEL SSDSC2KB240G7 U
----------------------------------------------------------------------------
`

		got := discoverStorcliDeviceIDsFromOutputs(eallOutput, sallOutput, 0)
		want := []string{"0:0", "0:1"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("device ids = %v, want %v", got, want)
		}
	})
}

func TestParseStorcliControllerVDInfo(t *testing.T) {
	t.Run("marks unsupported command as unusable", func(t *testing.T) {
		output := `{
"Controllers":[
{
	"Command Status" : {
		"Status" : "Failure",
		"Description" : "Un-supported command"
	}
}
]
}`

		got := parseStorcliControllerVDInfo(output)
		if got.Supported {
			t.Fatalf("expected unsupported VD info to be skipped")
		}
	})

	t.Run("keeps supported command output", func(t *testing.T) {
		output := `{"Controllers":[{"Response Data":{"VD LIST":[]}}]}`

		got := parseStorcliControllerVDInfo(output)
		if !got.Supported || got.Output != output {
			t.Fatalf("got %+v, want supported output to be preserved", got)
		}
	})
}

func TestParseStorcliControllerSnapshot(t *testing.T) {
	output := `
CLI Version = 007.0912.0000.0000 Dec 27, 2018
Operating system = Linux 3.10.0-693.el7.x86_64
Controller = 0
Status = Success
Description = Show Drive Information Succeeded.

Drive /c0/s0 :
============

----------------------------------------------------------------------------
EID:Slt DID State DG       Size Intf Med SED PI SeSz Model               Sp
----------------------------------------------------------------------------
 :0       0 Onln  -  222.585 GB SATA SSD N   N  512B INTEL SSDSC2KB240G7 U
----------------------------------------------------------------------------

Drive /c0/s0 State :
==================
Media Error Count = 0
Predictive Failure Count = 1

Drive /c0/s0 Device attributes :
==============================
Model Number = INTEL SSDSC2KB240G7
SN = PHYS729600SQ240AGN
Raw size = 223.570 GB [0x1bf244af Sectors]

Drive /c0/s4 :
============

----------------------------------------------------------------------------
EID:Slt DID State DG       Size Intf Med SED PI SeSz Model               Sp
----------------------------------------------------------------------------
 :4       2 UGood -  893.137 GB SATA SSD N   N  512B INTEL SSDSC2KB960G7 U
----------------------------------------------------------------------------

Drive /c0/s4 State :
==================
Media Error Count = 2
Predictive Failure Count = 3

Drive /c0/s4 Device attributes :
==============================
Model Number = INTEL SSDSC2KB960G7
SN = PHYS7410013X960CGN
Raw size = 894.252 GB [0x6fc81aaf Sectors]
`

	snapshot := parseStorcliControllerSnapshot(output, 0)
	wantIDs := []string{"0:0", "0:4"}
	if !reflect.DeepEqual(snapshot.DeviceIDs, wantIDs) {
		t.Fatalf("device ids = %v, want %v", snapshot.DeviceIDs, wantIDs)
	}

	disk0 := snapshot.Disks["0:0"]
	if disk0.State != "Onln" || disk0.PDType != "SATA" || disk0.MediaType != "SSD" {
		t.Fatalf("disk 0 basic fields = %+v", disk0)
	}
	if disk0.SerialNumber != "PHYS729600SQ240AGN" || disk0.Vendor != "INTEL" || disk0.Model != "SSDSC2KB240G7" {
		t.Fatalf("disk 0 identity fields = %+v", disk0)
	}
	if disk0.Capacity != "240 GB" || disk0.MediaError != "0" || disk0.PredictError != "1" {
		t.Fatalf("disk 0 metrics = %+v", disk0)
	}

	disk4 := snapshot.Disks["0:4"]
	if disk4.State != "UGood" || disk4.SerialNumber != "PHYS7410013X960CGN" || disk4.Capacity != "960 GB" {
		t.Fatalf("disk 4 fields = %+v", disk4)
	}
}

func TestFillStorcliLogicalVolumeNames(t *testing.T) {
	t.Run("assigns one shared logical volume to mirrored online members", func(t *testing.T) {
		disks := []Disk{
			{Name: "", CES: "0:0", State: "Onln", Capacity: "240 GB"},
			{Name: "", CES: "0:1", State: "Onln", Capacity: "240 GB"},
			{Name: "sdb", CES: "0:4", State: "UGood", Capacity: "960 GB"},
		}
		blockDevices := []storcliBlockDevice{
			{Name: "sda", SizeBytes: 239001149440, Type: "disk", Model: "Logical Volume", Vendor: "LSI"},
			{Name: "sdb", SizeBytes: 960197124096, Type: "disk", Model: "INTEL SSDSC2KB96", Vendor: "ATA"},
		}

		fillStorcliLogicalVolumeNames(disks, blockDevices)

		if disks[0].Name != "sda" || disks[1].Name != "sda" {
			t.Fatalf("logical volume names = %q, %q, want sda for both mirror members", disks[0].Name, disks[1].Name)
		}
	})

	t.Run("does not assign when multiple unused logical volumes remain", func(t *testing.T) {
		disks := []Disk{
			{Name: "", CES: "0:0", State: "Onln", Capacity: "240 GB"},
			{Name: "", CES: "0:1", State: "Onln", Capacity: "240 GB"},
		}
		blockDevices := []storcliBlockDevice{
			{Name: "sda", SizeBytes: 239001149440, Type: "disk", Model: "Logical Volume", Vendor: "LSI"},
			{Name: "sdb", SizeBytes: 239001149440, Type: "disk", Model: "Logical Volume", Vendor: "LSI"},
		}

		fillStorcliLogicalVolumeNames(disks, blockDevices)

		if disks[0].Name != "" || disks[1].Name != "" {
			t.Fatalf("expected names to remain empty, got %q and %q", disks[0].Name, disks[1].Name)
		}
	})
}
