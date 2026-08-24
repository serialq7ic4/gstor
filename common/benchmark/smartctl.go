package benchmark

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const MinimumSmartctlNVMeVersion = "7.0.2"

var smartctlVersionPattern = regexp.MustCompile(`(?m)^\s*smartctl\s+([0-9]+)\.([0-9]+)(?:\.([0-9]+))?\b`)

type SmartctlVersionStatus struct {
	Ready   bool
	Version string
	Warning string
}

type NVMeSmartctlCheck struct {
	HasNVMe bool
	Devices []string
	Ready   bool
	Version string
	Warning string
}

type semanticVersion struct {
	major int
	minor int
	patch int
}

func CheckNVMeSmartctlCompatibility(ctx context.Context, runner CommandRunner) (NVMeSmartctlCheck, error) {
	if runner == nil {
		runner = SystemRunner{}
	}

	result, err := runner.Run(ctx, "lsblk", "-dn", "-o", "NAME,TYPE")
	if err != nil {
		return NVMeSmartctlCheck{}, fmt.Errorf("failed to detect NVMe disks: %w: %s", err, result.Stderr)
	}
	devices := parseNVMeDiskNames(result.Stdout)
	check := NVMeSmartctlCheck{
		HasNVMe: len(devices) > 0,
		Devices: devices,
		Ready:   true,
	}
	if !check.HasNVMe {
		return check, nil
	}

	status, err := CheckSmartctlVersion(ctx, runner)
	if err != nil {
		check.Ready = false
		check.Warning = err.Error()
		return check, nil
	}
	check.Ready = status.Ready
	check.Version = status.Version
	check.Warning = status.Warning
	return check, nil
}

func CheckSmartctlVersion(ctx context.Context, runner CommandRunner) (SmartctlVersionStatus, error) {
	if runner == nil {
		runner = SystemRunner{}
	}
	result, err := runner.Run(ctx, "smartctl", "--version")
	if err != nil {
		return SmartctlVersionStatus{}, fmt.Errorf("failed to read smartctl version: %w: %s", err, result.Stderr)
	}
	version, err := parseSmartctlVersion(result.Stdout)
	if err != nil {
		return SmartctlVersionStatus{}, err
	}
	status := SmartctlVersionStatus{Version: version.String(), Ready: versionAtLeast(version, semanticVersion{major: 7, minor: 0, patch: 2})}
	if !status.Ready {
		status.Warning = fmt.Sprintf("NVMe disks require smartmontools >= %s, detected %s", MinimumSmartctlNVMeVersion, status.Version)
	}
	return status, nil
}

func parseNVMeDiskNames(output string) []string {
	seen := make(map[string]bool)
	var devices []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.EqualFold(fields[len(fields)-1], "disk") {
			continue
		}
		name := normalizeDiskSelector(fields[0])
		if !strings.HasPrefix(strings.ToLower(name), "nvme") || seen[name] {
			continue
		}
		seen[name] = true
		devices = append(devices, name)
	}
	sort.Strings(devices)
	return devices
}

func parseSmartctlVersion(output string) (semanticVersion, error) {
	matches := smartctlVersionPattern.FindStringSubmatch(output)
	if len(matches) == 0 {
		return semanticVersion{}, fmt.Errorf("failed to parse smartctl version from %q", strings.TrimSpace(output))
	}
	values := make([]int, 3)
	for idx := 0; idx < len(values); idx++ {
		value := matches[idx+1]
		if value == "" {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return semanticVersion{}, fmt.Errorf("failed to parse smartctl version %q: %w", value, err)
		}
		values[idx] = parsed
	}
	return semanticVersion{major: values[0], minor: values[1], patch: values[2]}, nil
}

func versionAtLeast(actual semanticVersion, minimum semanticVersion) bool {
	if actual.major != minimum.major {
		return actual.major > minimum.major
	}
	if actual.minor != minimum.minor {
		return actual.minor > minimum.minor
	}
	return actual.patch >= minimum.patch
}

func (v semanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func filterTargetsForNVMeSmartctl(targets []DiskTarget, check NVMeSmartctlCheck, explicit []string) ([]DiskTarget, []SkippedDisk, error) {
	if !check.HasNVMe || check.Ready {
		return targets, nil, nil
	}

	for _, selector := range explicit {
		name := normalizeDiskSelector(selector)
		if strings.HasPrefix(strings.ToLower(name), "nvme") {
			return nil, nil, fmt.Errorf("NVMe benchmark requires smartmontools >= %s; detected %s", MinimumSmartctlNVMeVersion, firstNonEmpty(check.Version, "unknown"))
		}
	}

	reason := firstNonEmpty(check.Warning, fmt.Sprintf("NVMe benchmark requires smartmontools >= %s", MinimumSmartctlNVMeVersion))
	filtered := make([]DiskTarget, 0, len(targets))
	var skipped []SkippedDisk
	for _, target := range targets {
		if !isNVMeTarget(target) {
			filtered = append(filtered, target)
			continue
		}
		skipped = append(skipped, SkippedDisk{Name: target.Name, DevicePath: target.DevicePath, Reason: reason})
	}
	return filtered, skipped, nil
}

func isNVMeTarget(target DiskTarget) bool {
	return strings.EqualFold(strings.TrimSpace(target.InterfaceType), "NVME") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(target.Name)), "nvme")
}
