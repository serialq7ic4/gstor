package benchmark

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeCommandRunner struct {
	results []fakeCommandResult
	calls   [][]string
}

type fakeCommandResult struct {
	result CommandResult
	err    error
}

func (r *fakeCommandRunner) Run(ctx context.Context, name string, args ...string) (CommandResult, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if len(r.results) == 0 {
		return CommandResult{}, nil
	}
	result := r.results[0]
	r.results = r.results[1:]
	return result.result, result.err
}

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

func TestDiscoverEligibleDisksFromLSBLKPairsSupportsCentOS7Output(t *testing.T) {
	input := []byte(`NAME="sda" KNAME="sda" TYPE="disk" FSTYPE="" MOUNTPOINT="" ROTA="1" TRAN="" MODEL="PERC H730P" SERIAL="" SIZE="239444426752" RO="0" PKNAME="" REV="" VENDOR="DELL"
NAME="sda1" KNAME="sda1" TYPE="part" FSTYPE="xfs" MOUNTPOINT="/boot" ROTA="1" TRAN="" MODEL="" SERIAL="" SIZE="1073741824" RO="0" PKNAME="sda" REV="" VENDOR=""
NAME="sdb" KNAME="sdb" TYPE="disk" FSTYPE="" MOUNTPOINT="" ROTA="1" TRAN="" MODEL="PERC H730P" SERIAL="SDB123" SIZE="959656755200" RO="0" PKNAME="" REV="1.0" VENDOR="DELL"
NAME="sdc" KNAME="sdc" TYPE="disk" FSTYPE="" MOUNTPOINT="" ROTA="1" TRAN="" MODEL="PERC H730P" SERIAL="SDC123" SIZE="959656755200" RO="0" PKNAME="" REV="1.0" VENDOR="DELL"
NAME="sdc1" KNAME="sdc1" TYPE="part" FSTYPE="ext4" MOUNTPOINT="/data2" ROTA="1" TRAN="" MODEL="" SERIAL="" SIZE="959654658048" RO="0" PKNAME="sdc" REV="" VENDOR=""`)

	targets, skipped, err := DiscoverEligibleDisksFromLSBLKPairs(input, []string{"sdb"})
	if err != nil {
		t.Fatalf("DiscoverEligibleDisksFromLSBLKPairs returned error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("explicit eligible disk should not produce skips: %+v", skipped)
	}
	if len(targets) != 1 {
		t.Fatalf("targets = %+v, want one target", targets)
	}
	if targets[0].Name != "sdb" || targets[0].DevicePath != "/dev/sdb" {
		t.Fatalf("unexpected target identity: %+v", targets[0])
	}
	if targets[0].MediaType != "HDD" || targets[0].InterfaceType != "UNKNOWN" {
		t.Fatalf("unexpected target media/interface: %+v", targets[0])
	}

	allTargets, allSkipped, err := DiscoverEligibleDisksFromLSBLKPairs(input, nil)
	if err != nil {
		t.Fatalf("DiscoverEligibleDisksFromLSBLKPairs all returned error: %v", err)
	}
	if len(allTargets) != 1 || allTargets[0].Name != "sdb" {
		t.Fatalf("all targets = %+v, want only sdb", allTargets)
	}
	wantSkipped := map[string]string{
		"sda": "has child block devices",
		"sdc": "has child block devices",
	}
	for _, skippedDisk := range allSkipped {
		if wantSkipped[skippedDisk.Name] != skippedDisk.Reason {
			t.Fatalf("skip reason for %s = %q, want %q", skippedDisk.Name, skippedDisk.Reason, wantSkipped[skippedDisk.Name])
		}
		delete(wantSkipped, skippedDisk.Name)
	}
	if len(wantSkipped) != 0 {
		t.Fatalf("missing skipped disks: %+v", wantSkipped)
	}
}

func TestDiscoverEligibleDisksFallsBackToPairsWhenJSONUnsupported(t *testing.T) {
	runner := &fakeCommandRunner{
		results: []fakeCommandResult{
			{
				result: CommandResult{Stderr: "lsblk: invalid option -- 'J'", ExitCode: 1},
				err:    errors.New("exit status 1"),
			},
			{
				result: CommandResult{Stdout: `NAME="sdb" KNAME="sdb" TYPE="disk" FSTYPE="" MOUNTPOINT="" ROTA="1" TRAN="" MODEL="PERC H730P" SERIAL="SDB123" SIZE="959656755200" RO="0" PKNAME="" REV="1.0" VENDOR="DELL"`},
			},
		},
	}

	targets, skipped, err := DiscoverEligibleDisks(context.Background(), Inspector{
		Runner:       runner,
		SysBlockRoot: t.TempDir(),
	}, []string{"sdb"})
	if err != nil {
		t.Fatalf("DiscoverEligibleDisks returned error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("skipped = %+v, want none", skipped)
	}
	if len(targets) != 1 || targets[0].Name != "sdb" {
		t.Fatalf("targets = %+v, want sdb", targets)
	}

	wantCalls := [][]string{
		{"lsblk", "-J", "-O", "-b"},
		{"lsblk", "-b", "-P", "-o", "NAME,KNAME,TYPE,FSTYPE,MOUNTPOINT,ROTA,TRAN,MODEL,SERIAL,SIZE,RO,PKNAME,REV,VENDOR"},
		{"findmnt", "-rn", "-o", "SOURCE", "/"},
		{"findmnt", "-rn", "-o", "SOURCE", "/boot"},
		{"findmnt", "-rn", "-o", "SOURCE", "/boot/efi"},
		{"blkid", "/dev/sdb"},
		{"findmnt", "-rn", "--source", "/dev/sdb"},
		{"fuser", "/dev/sdb"},
		{"pvs", "--noheadings", "-o", "pv_name"},
		{"lvs", "--noheadings", "-o", "devices"},
		{"vgs", "--noheadings", "-o", "vg_name"},
		{"mdadm", "--detail", "--scan"},
		{"mdadm", "--examine", "--brief", "/dev/sdb"},
		{"dmsetup", "deps", "-o", "devname"},
		{"multipath", "-ll"},
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, wantCalls)
	}
}

type scriptedCommandRunner struct {
	handlers map[string]func(context.Context) (CommandResult, error)
	calls    []string
}

func (r *scriptedCommandRunner) Run(ctx context.Context, name string, args ...string) (CommandResult, error) {
	key := strings.TrimSpace(name + " " + strings.Join(args, " "))
	r.calls = append(r.calls, key)
	if handler, ok := r.handlers[key]; ok {
		return handler(ctx)
	}
	return CommandResult{}, nil
}

func newSafetyTestInspector(t *testing.T, runner CommandRunner, disk string) Inspector {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"holders", "slaves"} {
		if err := os.MkdirAll(filepath.Join(root, disk, dir), 0755); err != nil {
			t.Fatalf("mkdir sys block fixture: %v", err)
		}
	}
	return Inspector{Runner: runner, SysBlockRoot: root}
}

func TestExtraSafetyRejectsRootAndBootDisks(t *testing.T) {
	runner := &scriptedCommandRunner{handlers: map[string]func(context.Context) (CommandResult, error){
		"findmnt -rn -o SOURCE /": func(context.Context) (CommandResult, error) {
			return CommandResult{Stdout: "/dev/sdb1"}, nil
		},
	}}

	inspector := newSafetyTestInspector(t, runner, "sdb")
	reason := inspector.extraSafetyReason(context.Background(), DiskTarget{Name: "sdb", DevicePath: "/dev/sdb"})
	if reason != "contains root filesystem" {
		t.Fatalf("root disk reason = %q, want contains root filesystem", reason)
	}
}

func TestExtraSafetyRejectsStorageStackMembers(t *testing.T) {
	tests := []struct {
		name    string
		command string
		stdout  string
		want    string
	}{
		{
			name:    "lvm physical volume",
			command: "pvs --noheadings -o pv_name",
			stdout:  "  /dev/sdb\n",
			want:    "is LVM physical volume",
		},
		{
			name:    "mdraid member",
			command: "mdadm --examine --brief /dev/sdb",
			stdout:  "/dev/sdb: ARRAY /dev/md/0 metadata=1.2 UUID=abc\n",
			want:    "has mdraid metadata",
		},
		{
			name:    "device mapper dependency",
			command: "dmsetup deps -o devname",
			stdout:  "vg-root: 1 dependencies : (sdb)\n",
			want:    "is used by device-mapper",
		},
		{
			name:    "multipath member",
			command: "multipath -ll",
			stdout:  "mpatha dm-0\n`- 0:0:0:0 sdb 8:16 active ready running\n",
			want:    "is used by multipath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &scriptedCommandRunner{handlers: map[string]func(context.Context) (CommandResult, error){
				tt.command: func(context.Context) (CommandResult, error) {
					return CommandResult{Stdout: tt.stdout}, nil
				},
			}}
			inspector := newSafetyTestInspector(t, runner, "sdb")
			reason := inspector.extraSafetyReason(context.Background(), DiskTarget{Name: "sdb", DevicePath: "/dev/sdb"})
			if reason != tt.want {
				t.Fatalf("reason = %q, want %q; calls=%v", reason, tt.want, runner.calls)
			}
		})
	}
}

func TestExtraSafetyUsesPerCheckTimeout(t *testing.T) {
	runner := &scriptedCommandRunner{handlers: map[string]func(context.Context) (CommandResult, error){
		"blkid /dev/sdb": func(ctx context.Context) (CommandResult, error) {
			if _, ok := ctx.Deadline(); !ok {
				return CommandResult{}, nil
			}
			return CommandResult{}, context.DeadlineExceeded
		},
	}}
	inspector := newSafetyTestInspector(t, runner, "sdb")

	reason := inspector.extraSafetyReason(context.Background(), DiskTarget{Name: "sdb", DevicePath: "/dev/sdb"})
	if reason != "safety check timed out: blkid" {
		t.Fatalf("timeout reason = %q, want safety check timed out: blkid", reason)
	}
}
