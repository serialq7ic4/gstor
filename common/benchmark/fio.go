package benchmark

import (
	"fmt"
	"strings"
)

type FIOCaseResult struct {
	CaseID     string         `json:"case_id"`
	CaseOrder  int            `json:"case_order"`
	Parameters map[string]any `json:"parameters,omitempty"`
	Metrics    map[string]any `json:"metrics,omitempty"`
	RawFIOJSON map[string]any `json:"raw_fio_json,omitempty"`
}

func RenderFIOJob(profile Profile, disk DiskTarget) (string, error) {
	if strings.TrimSpace(disk.DevicePath) == "" {
		return "", fmt.Errorf("disk device path is required")
	}
	if len(profile.Cases) == 0 {
		return "", fmt.Errorf("profile %s_%s has no cases", profile.Name, profile.Version)
	}

	var builder strings.Builder
	builder.WriteString("[global]\n")
	builder.WriteString("ioengine=libaio\n")
	builder.WriteString("direct=1\n")
	builder.WriteString("time_based=1\n")
	builder.WriteString("runtime=60\n")
	builder.WriteString("ramp_time=10\n")
	builder.WriteString("randrepeat=0\n")
	builder.WriteString("refill_buffers=1\n")
	builder.WriteString("group_reporting=1\n")
	builder.WriteString("output-format=json\n")
	builder.WriteString("percentile_list=50:90:95:99:99.9\n\n")

	for _, c := range profile.Cases {
		pressure := PressureFor(disk, c.Phase)
		builder.WriteString("[")
		builder.WriteString(c.ID)
		builder.WriteString("]\n")
		builder.WriteString("stonewall\n")
		builder.WriteString("filename=")
		builder.WriteString(disk.DevicePath)
		builder.WriteString("\n")
		builder.WriteString("rw=")
		builder.WriteString(c.RW)
		builder.WriteString("\n")
		builder.WriteString("bs=")
		builder.WriteString(c.BlockSize)
		builder.WriteString("\n")
		_, _ = fmt.Fprintf(&builder, "numjobs=%d\n", pressure.NumJobs)
		_, _ = fmt.Fprintf(&builder, "iodepth=%d\n", pressure.IODepth)
		if c.RWMixRead > 0 {
			_, _ = fmt.Fprintf(&builder, "rwmixread=%d\n", c.RWMixRead)
		}
		if c.Offset != "" {
			builder.WriteString("offset=")
			builder.WriteString(c.Offset)
			builder.WriteString("\n")
		}
		if c.Size != "" {
			builder.WriteString("size=")
			builder.WriteString(c.Size)
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func ValidateFIOCases(profile Profile, cases []FIOCaseResult) error {
	if len(cases) != len(profile.Cases) {
		return fmt.Errorf("profile %s_%s requires %d cases, received %d", profile.Name, profile.Version, len(profile.Cases), len(cases))
	}

	expected := make(map[string]CaseDefinition, len(profile.Cases))
	for _, c := range profile.Cases {
		expected[c.ID] = c
	}

	seen := make(map[string]bool, len(cases))
	for _, c := range cases {
		definition, ok := expected[c.CaseID]
		if !ok {
			return fmt.Errorf("profile %s_%s received unknown case %q", profile.Name, profile.Version, c.CaseID)
		}
		if seen[c.CaseID] {
			return fmt.Errorf("profile %s_%s received duplicate case %q", profile.Name, profile.Version, c.CaseID)
		}
		if c.CaseOrder != definition.Order {
			return fmt.Errorf("case %q order = %d, want %d", c.CaseID, c.CaseOrder, definition.Order)
		}
		seen[c.CaseID] = true
	}

	for _, c := range profile.Cases {
		if !seen[c.ID] {
			return fmt.Errorf("profile %s_%s missing case %q", profile.Name, profile.Version, c.ID)
		}
	}

	return nil
}
