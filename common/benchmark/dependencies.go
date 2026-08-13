package benchmark

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var requiredDependencies = []string{"fio", "lsblk", "blkid", "findmnt", "fuser", "smartctl"}

type Probe interface {
	LookPath(name string) (string, error)
	EUID() int
}

type SystemProbe struct{}

func (SystemProbe) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (SystemProbe) EUID() int {
	return os.Geteuid()
}

type DependencyStatus struct {
	Name      string `json:"name"`
	Required  bool   `json:"required"`
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
	Error     string `json:"error,omitempty"`
}

type RequirementReport struct {
	Ready        bool               `json:"ready"`
	Root         bool               `json:"root"`
	Dependencies []DependencyStatus `json:"dependencies"`
}

func CheckRequirements(probe Probe) RequirementReport {
	report := RequirementReport{
		Root: probe.EUID() == 0,
	}

	for _, name := range requiredDependencies {
		path, err := probe.LookPath(name)
		status := DependencyStatus{
			Name:     name,
			Required: true,
		}
		if err != nil {
			status.Error = err.Error()
		} else {
			status.Available = true
			status.Path = path
		}
		report.Dependencies = append(report.Dependencies, status)
	}

	report.Ready = report.Root && len(report.MissingNames()) == 0
	return report
}

func (r RequirementReport) MissingNames() []string {
	var missing []string
	for _, dep := range r.Dependencies {
		if dep.Required && !dep.Available {
			missing = append(missing, dep.Name)
		}
	}
	if !r.Root {
		missing = append(missing, "root")
	}
	return missing
}

func (r RequirementReport) Error() error {
	if r.Ready {
		return nil
	}
	return fmt.Errorf("benchmark requirements not met: %s", strings.Join(r.MissingNames(), ", "))
}
