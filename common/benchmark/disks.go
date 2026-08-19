package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type SkippedDisk struct {
	Name       string `json:"name"`
	DevicePath string `json:"device_path,omitempty"`
	Reason     string `json:"reason"`
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (CommandResult, error)
}

type SystemRunner struct{}

func (SystemRunner) Run(ctx context.Context, name string, args ...string) (CommandResult, error) {
	command := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := command.Start(); err != nil {
		return CommandResult{ExitCode: -1}, err
	}

	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()

	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		pid := command.Process.Pid
		_ = syscall.Kill(-pid, syscall.SIGTERM)

		timer := time.NewTimer(10 * time.Second)
		select {
		case err = <-done:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			err = <-done
		}
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	}

	result := CommandResult{
		Stdout: strings.TrimRight(stdout.String(), "\n"),
		Stderr: strings.TrimRight(stderr.String(), "\n"),
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result, err
	}
	result.ExitCode = 0
	return result, nil
}

type Inspector struct {
	Runner             CommandRunner
	SysBlockRoot       string
	SafetyCheckTimeout time.Duration
}

func NewSystemInspector() Inspector {
	return Inspector{
		Runner:             SystemRunner{},
		SysBlockRoot:       "/sys/block",
		SafetyCheckTimeout: 5 * time.Second,
	}
}

func DiscoverEligibleDisks(ctx context.Context, inspector Inspector, explicit []string) ([]DiskTarget, []SkippedDisk, error) {
	if inspector.Runner == nil {
		inspector.Runner = SystemRunner{}
	}
	if inspector.SysBlockRoot == "" {
		inspector.SysBlockRoot = "/sys/block"
	}

	result, err := inspector.Runner.Run(ctx, "lsblk", "-J", "-O", "-b")
	if err != nil {
		pairResult, pairErr := inspector.Runner.Run(ctx, "lsblk", "-b", "-P", "-o", "NAME,KNAME,TYPE,FSTYPE,MOUNTPOINT,ROTA,TRAN,MODEL,SERIAL,SIZE,RO,PKNAME,REV,VENDOR")
		if pairErr != nil {
			return nil, nil, fmt.Errorf("failed to run lsblk JSON: %w: %s; failed to run lsblk pairs: %w: %s", err, result.Stderr, pairErr, pairResult.Stderr)
		}
		return eligibleDisksWithExtraSafety(ctx, inspector, []byte(pairResult.Stdout), explicit, DiscoverEligibleDisksFromLSBLKPairs)
	}

	return eligibleDisksWithExtraSafety(ctx, inspector, []byte(result.Stdout), explicit, DiscoverEligibleDisksFromLSBLK)
}

func eligibleDisksWithExtraSafety(
	ctx context.Context,
	inspector Inspector,
	data []byte,
	explicit []string,
	parser func([]byte, []string) ([]DiskTarget, []SkippedDisk, error),
) ([]DiskTarget, []SkippedDisk, error) {
	targets, skipped, err := parser(data, explicit)
	if err != nil {
		return nil, skipped, err
	}

	var safeTargets []DiskTarget
	for _, target := range targets {
		if reason := inspector.extraSafetyReason(ctx, target); reason != "" {
			skipped = append(skipped, SkippedDisk{Name: target.Name, DevicePath: target.DevicePath, Reason: reason})
			continue
		}
		safeTargets = append(safeTargets, target)
	}

	return safeTargets, skipped, nil
}

type lsblkDocument struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name        string        `json:"name"`
	Path        string        `json:"path"`
	Type        string        `json:"type"`
	Tran        string        `json:"tran"`
	Rota        *bool         `json:"rota"`
	Size        json.Number   `json:"size"`
	Serial      string        `json:"serial"`
	Vendor      string        `json:"vendor"`
	Model       string        `json:"model"`
	Firmware    string        `json:"rev"`
	FSType      string        `json:"fstype"`
	MountPoint  any           `json:"mountpoint"`
	MountPoints []any         `json:"mountpoints"`
	ReadOnly    any           `json:"ro"`
	Children    []lsblkDevice `json:"children"`
}

func DiscoverEligibleDisksFromLSBLK(data []byte, explicit []string) ([]DiskTarget, []SkippedDisk, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var document lsblkDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, nil, fmt.Errorf("failed to parse lsblk JSON: %w", err)
	}

	return eligibleDisksFromDevices(document.BlockDevices, explicit)
}

func DiscoverEligibleDisksFromLSBLKPairs(data []byte, explicit []string) ([]DiskTarget, []SkippedDisk, error) {
	rows := strings.Split(strings.TrimSpace(string(data)), "\n")
	devicesByName := make(map[string]lsblkDevice)
	var diskOrder []string
	childrenByParent := make(map[string][]lsblkDevice)

	for _, row := range rows {
		fields := parseLSBLKPairs(row)
		if len(fields) == 0 {
			continue
		}
		device := lsblkDevice{
			Name:       firstNonEmpty(fields["KNAME"], fields["NAME"]),
			Type:       fields["TYPE"],
			Tran:       fields["TRAN"],
			Rota:       parseBoolPointer(fields["ROTA"]),
			Size:       json.Number(fields["SIZE"]),
			Serial:     fields["SERIAL"],
			Vendor:     fields["VENDOR"],
			Model:      fields["MODEL"],
			Firmware:   fields["REV"],
			FSType:     fields["FSTYPE"],
			MountPoint: fields["MOUNTPOINT"],
			ReadOnly:   fields["RO"],
		}
		if device.Name == "" {
			continue
		}

		parent := fields["PKNAME"]
		if device.Type == "disk" {
			devicesByName[device.Name] = device
			diskOrder = append(diskOrder, device.Name)
			continue
		}
		if parent != "" {
			childrenByParent[parent] = append(childrenByParent[parent], device)
		}
	}

	devices := make([]lsblkDevice, 0, len(diskOrder))
	for _, name := range diskOrder {
		device := devicesByName[name]
		device.Children = childrenByParent[name]
		devices = append(devices, device)
	}

	return eligibleDisksFromDevices(devices, explicit)
}

var lsblkPairPattern = regexp.MustCompile(`([A-Z0-9:_-]+)="([^"]*)"`)

func parseLSBLKPairs(row string) map[string]string {
	fields := make(map[string]string)
	for _, match := range lsblkPairPattern.FindAllStringSubmatch(row, -1) {
		fields[match[1]] = match[2]
	}
	return fields
}

func eligibleDisksFromDevices(devices []lsblkDevice, explicit []string) ([]DiskTarget, []SkippedDisk, error) {
	explicitSet := make(map[string]bool, len(explicit))
	for _, disk := range explicit {
		normalized := normalizeDiskSelector(disk)
		if normalized != "" {
			explicitSet[normalized] = false
		}
	}

	var targets []DiskTarget
	var skipped []SkippedDisk
	for _, device := range devices {
		if len(explicitSet) > 0 {
			_, wantName := explicitSet[device.Name]
			_, wantPath := explicitSet[normalizeDiskSelector(device.Path)]
			if !wantName && !wantPath {
				continue
			}
			if wantName {
				explicitSet[device.Name] = true
			}
			if wantPath {
				explicitSet[normalizeDiskSelector(device.Path)] = true
			}
		}

		if reason := unsafeLSBLKReason(device); reason != "" {
			skipped = append(skipped, SkippedDisk{Name: device.Name, DevicePath: devicePath(device), Reason: reason})
			continue
		}
		targets = append(targets, diskTargetFromLSBLK(device))
	}

	for selector, found := range explicitSet {
		if !found {
			return nil, skipped, fmt.Errorf("explicit disk %q was not found by lsblk", selector)
		}
	}

	return targets, skipped, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseBoolPointer(value string) *bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "1", "true":
		parsed := true
		return &parsed
	case "0", "false":
		parsed := false
		return &parsed
	default:
		return nil
	}
}

func normalizeDiskSelector(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "/dev/") {
		return filepath.Base(value)
	}
	return value
}

func unsafeLSBLKReason(device lsblkDevice) string {
	if device.Type != "disk" {
		return fmt.Sprintf("unsupported block device type %s", device.Type)
	}
	if isVirtualDiskName(device.Name) {
		return "virtual block device"
	}
	if readOnly(device.ReadOnly) {
		return "read-only block device"
	}
	if len(device.Children) > 0 {
		return "has child block devices"
	}
	if hasMountpoint(device) {
		return "has mountpoint"
	}
	if strings.TrimSpace(device.FSType) != "" {
		return "has filesystem signature"
	}
	return ""
}

func isVirtualDiskName(name string) bool {
	for _, prefix := range []string{"loop", "ram", "rom", "zram"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func readOnly(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed != 0
	case json.Number:
		i, err := typed.Int64()
		return err == nil && i != 0
	case string:
		return typed == "1" || strings.EqualFold(typed, "true")
	default:
		return false
	}
}

func hasMountpoint(device lsblkDevice) bool {
	if mountpointValue(device.MountPoint) {
		return true
	}
	for _, mountpoint := range device.MountPoints {
		if mountpointValue(mountpoint) {
			return true
		}
	}
	return false
}

func mountpointValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	default:
		return false
	}
}

func diskTargetFromLSBLK(device lsblkDevice) DiskTarget {
	capacityBytes := parseJSONNumberUint64(device.Size)
	return DiskTarget{
		Name:          device.Name,
		DevicePath:    devicePath(device),
		SerialNumber:  strings.TrimSpace(device.Serial),
		Vendor:        strings.TrimSpace(device.Vendor),
		Model:         strings.TrimSpace(device.Model),
		Firmware:      strings.TrimSpace(device.Firmware),
		CapacityBytes: capacityBytes,
		Capacity:      formatBytes(capacityBytes),
		MediaType:     mediaTypeFromRota(device.Rota),
		InterfaceType: interfaceTypeFromTran(device.Tran, device.Name),
	}
}

func devicePath(device lsblkDevice) string {
	if strings.TrimSpace(device.Path) != "" {
		return strings.TrimSpace(device.Path)
	}
	return "/dev/" + device.Name
}

func parseJSONNumberUint64(number json.Number) uint64 {
	if strings.TrimSpace(number.String()) == "" {
		return 0
	}
	value, err := strconv.ParseUint(number.String(), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func mediaTypeFromRota(rota *bool) string {
	if rota == nil {
		return "UNKNOWN"
	}
	if *rota {
		return "HDD"
	}
	return "SSD"
}

func interfaceTypeFromTran(tran string, name string) string {
	switch strings.ToLower(strings.TrimSpace(tran)) {
	case "sata":
		return "SATA"
	case "sas":
		return "SAS"
	case "nvme":
		return "NVME"
	}
	if strings.HasPrefix(name, "nvme") {
		return "NVME"
	}
	return "UNKNOWN"
}

func formatBytes(bytes uint64) string {
	if bytes == 0 {
		return ""
	}
	const unit = uint64(1000)
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := unit, 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (i Inspector) extraSafetyReason(ctx context.Context, target DiskTarget) string {
	if reason := nonEmptySysDir(filepath.Join(i.SysBlockRoot, target.Name, "holders"), "has holder devices"); reason != "" {
		return reason
	}
	if reason := nonEmptySysDir(filepath.Join(i.SysBlockRoot, target.Name, "slaves"), "has slave devices"); reason != "" {
		return reason
	}
	if reason := i.mountedSystemDiskReason(ctx, target); reason != "" {
		return reason
	}
	if reason := i.commandOutputReason(ctx, "has filesystem signature", "blkid", target.DevicePath); reason != "" {
		return reason
	}
	if reason := i.commandOutputReason(ctx, "has mountpoint", "findmnt", "-rn", "--source", target.DevicePath); reason != "" {
		return reason
	}
	if reason := i.commandOutputReason(ctx, "device is in use", "fuser", target.DevicePath); reason != "" {
		return reason
	}
	if reason := i.storageStackReason(ctx, target); reason != "" {
		return reason
	}
	return ""
}

func (i Inspector) mountedSystemDiskReason(ctx context.Context, target DiskTarget) string {
	checks := []struct {
		mountpoint string
		reason     string
	}{
		{mountpoint: "/", reason: "contains root filesystem"},
		{mountpoint: "/boot", reason: "contains boot filesystem"},
		{mountpoint: "/boot/efi", reason: "contains boot filesystem"},
	}

	for _, check := range checks {
		result, timedOut := i.runSafetyCheck(ctx, "findmnt", "-rn", "-o", "SOURCE", check.mountpoint)
		if timedOut {
			return "safety check timed out: findmnt"
		}
		if sourceBelongsToDisk(result.Stdout, target.Name) {
			return check.reason
		}
	}
	return ""
}

func (i Inspector) storageStackReason(ctx context.Context, target DiskTarget) string {
	checks := []struct {
		reason string
		name   string
		args   []string
	}{
		{reason: "is LVM physical volume", name: "pvs", args: []string{"--noheadings", "-o", "pv_name"}},
		{reason: "is LVM logical volume backing device", name: "lvs", args: []string{"--noheadings", "-o", "devices"}},
		{reason: "is listed in volume groups", name: "vgs", args: []string{"--noheadings", "-o", "vg_name"}},
		{reason: "is used by mdraid", name: "mdadm", args: []string{"--detail", "--scan"}},
		{reason: "has mdraid metadata", name: "mdadm", args: []string{"--examine", "--brief", target.DevicePath}},
		{reason: "is used by device-mapper", name: "dmsetup", args: []string{"deps", "-o", "devname"}},
		{reason: "is used by multipath", name: "multipath", args: []string{"-ll"}},
	}

	for _, check := range checks {
		result, timedOut := i.runSafetyCheck(ctx, check.name, check.args...)
		if timedOut {
			return "safety check timed out: " + check.name
		}
		output := result.Stdout + "\n" + result.Stderr
		if check.name == "mdadm" && len(check.args) >= 2 && check.args[0] == "--examine" && strings.Contains(output, "No md superblock") {
			continue
		}
		if textReferencesDisk(output, target) {
			return check.reason
		}
	}
	return ""
}

func (i Inspector) commandOutputReason(ctx context.Context, reason string, name string, args ...string) string {
	result, timedOut := i.runSafetyCheck(ctx, name, args...)
	if timedOut {
		return "safety check timed out: " + name
	}
	if strings.TrimSpace(result.Stdout) != "" || strings.TrimSpace(result.Stderr) != "" {
		return reason
	}
	return ""
}

func (i Inspector) runSafetyCheck(ctx context.Context, name string, args ...string) (CommandResult, bool) {
	timeout := i.SafetyCheckTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := i.Runner.Run(checkCtx, name, args...)
	timedOut := errors.Is(err, context.DeadlineExceeded) || errors.Is(checkCtx.Err(), context.DeadlineExceeded)
	return result, timedOut
}

func nonEmptySysDir(path string, reason string) string {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		return "safety check failed: " + err.Error()
	}
	if len(entries) > 0 {
		return reason
	}
	return ""
}

func sourceBelongsToDisk(source string, diskName string) bool {
	source = strings.TrimSpace(source)
	if source == "" || diskName == "" {
		return false
	}
	base := filepath.Base(strings.TrimPrefix(source, "/dev/"))
	if base == diskName {
		return true
	}
	if !strings.HasPrefix(base, diskName) {
		return false
	}
	suffix := strings.TrimPrefix(base, diskName)
	if suffix == "" {
		return true
	}
	suffix = strings.TrimPrefix(suffix, "p")
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func textReferencesDisk(output string, target DiskTarget) bool {
	if strings.TrimSpace(output) == "" {
		return false
	}
	if target.DevicePath != "" && strings.Contains(output, target.DevicePath) {
		return true
	}
	split := func(r rune) bool {
		return r != '/' && r != '_' && r != '-' && r != '.' && r != ':' &&
			(r < '0' || r > '9') &&
			(r < 'A' || r > 'Z') &&
			(r < 'a' || r > 'z')
	}
	for _, token := range strings.FieldsFunc(output, split) {
		token = strings.Trim(token, "`'\"(),;")
		if sourceBelongsToDisk(token, target.Name) {
			return true
		}
	}
	return false
}
