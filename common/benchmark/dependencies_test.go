package benchmark

import (
	"errors"
	"testing"
)

type fakeProbe struct {
	paths map[string]string
	euid  int
}

func (p fakeProbe) LookPath(name string) (string, error) {
	if path, ok := p.paths[name]; ok {
		return path, nil
	}
	return "", errors.New("not found")
}

func (p fakeProbe) EUID() int {
	return p.euid
}

func TestCheckRequirementsRequiresRootAndTools(t *testing.T) {
	report := CheckRequirements(fakeProbe{
		paths: map[string]string{
			"fio":      "/usr/bin/fio",
			"lsblk":    "/usr/bin/lsblk",
			"blkid":    "/usr/sbin/blkid",
			"findmnt":  "/usr/bin/findmnt",
			"fuser":    "/usr/bin/fuser",
			"smartctl": "/usr/sbin/smartctl",
		},
		euid: 0,
	})

	if !report.Ready {
		t.Fatalf("report should be ready: %+v", report)
	}
	if len(report.Dependencies) != 6 {
		t.Fatalf("dependency count = %d, want 6", len(report.Dependencies))
	}
}

func TestCheckRequirementsReportsMissingToolsAndNonRoot(t *testing.T) {
	report := CheckRequirements(fakeProbe{
		paths: map[string]string{
			"fio":   "/usr/bin/fio",
			"lsblk": "/usr/bin/lsblk",
		},
		euid: 1000,
	})

	if report.Ready {
		t.Fatal("report should not be ready")
	}
	if report.Root {
		t.Fatal("root should be false for non-root euid")
	}
	if report.Error() == nil {
		t.Fatal("report should expose an aggregate error")
	}

	missing := report.MissingNames()
	want := []string{"blkid", "findmnt", "fuser", "smartctl", "root"}
	if len(missing) != len(want) {
		t.Fatalf("missing = %v, want %v", missing, want)
	}
	for i := range want {
		if missing[i] != want[i] {
			t.Fatalf("missing = %v, want %v", missing, want)
		}
	}
}
