package benchmark

import "testing"

func TestDiscoverEligibleDisksFromLSBLKExcludesUnsafeDevices(t *testing.T) {
	input := []byte(`{
  "blockdevices": [
    {
      "name": "sda",
      "path": "/dev/sda",
      "type": "disk",
      "tran": "sata",
      "rota": true,
      "size": 1000000000,
      "mountpoints": [null],
      "children": [
        {"name": "sda1", "type": "part", "fstype": "ext4", "mountpoints": ["/"]}
      ]
    },
    {
      "name": "sdb",
      "path": "/dev/sdb",
      "type": "disk",
      "tran": "sata",
      "rota": true,
      "size": 2000000000,
      "fstype": "xfs",
      "mountpoints": [null]
    },
    {
      "name": "loop0",
      "path": "/dev/loop0",
      "type": "loop",
      "size": 3000000000
    },
    {
      "name": "sdc",
      "path": "/dev/sdc",
      "type": "disk",
      "tran": "sas",
      "rota": true,
      "size": 4000000000,
      "serial": "SAS123",
      "vendor": "SEAGATE",
      "model": "ST4000"
    },
    {
      "name": "nvme0n1",
      "path": "/dev/nvme0n1",
      "type": "disk",
      "tran": "nvme",
      "rota": false,
      "size": 5000000000,
      "serial": "NVME123",
      "model": "PM9A3"
    }
  ]
}`)

	targets, skipped, err := DiscoverEligibleDisksFromLSBLK(input, nil)
	if err != nil {
		t.Fatalf("DiscoverEligibleDisksFromLSBLK returned error: %v", err)
	}

	if len(targets) != 2 {
		t.Fatalf("eligible targets = %+v, want 2", targets)
	}
	if targets[0].Name != "sdc" || targets[0].MediaType != "HDD" || targets[0].InterfaceType != "SAS" {
		t.Fatalf("unexpected first target: %+v", targets[0])
	}
	if targets[1].Name != "nvme0n1" || targets[1].MediaType != "SSD" || targets[1].InterfaceType != "NVME" {
		t.Fatalf("unexpected second target: %+v", targets[1])
	}

	wantSkipped := map[string]string{
		"sda":   "has child block devices",
		"sdb":   "has filesystem signature",
		"loop0": "unsupported block device type loop",
	}
	if len(skipped) != len(wantSkipped) {
		t.Fatalf("skipped = %+v, want %d entries", skipped, len(wantSkipped))
	}
	for _, s := range skipped {
		if wantSkipped[s.Name] != s.Reason {
			t.Fatalf("skip reason for %s = %q, want %q", s.Name, s.Reason, wantSkipped[s.Name])
		}
	}
}

func TestDiscoverEligibleDisksFromLSBLKResolvesExplicitDisks(t *testing.T) {
	input := []byte(`{
  "blockdevices": [
    {"name": "sdb", "path": "/dev/sdb", "type": "disk", "tran": "sata", "rota": false, "size": 2000000000},
    {"name": "sdc", "path": "/dev/sdc", "type": "disk", "tran": "sas", "rota": true, "size": 4000000000}
  ]
}`)

	targets, skipped, err := DiscoverEligibleDisksFromLSBLK(input, []string{"/dev/sdc"})
	if err != nil {
		t.Fatalf("DiscoverEligibleDisksFromLSBLK returned error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("explicit eligible disk should not produce skips: %+v", skipped)
	}
	if len(targets) != 1 || targets[0].Name != "sdc" {
		t.Fatalf("targets = %+v, want only sdc", targets)
	}

	_, _, err = DiscoverEligibleDisksFromLSBLK(input, []string{"missing"})
	if err == nil {
		t.Fatal("missing explicit disk should return error")
	}
}
