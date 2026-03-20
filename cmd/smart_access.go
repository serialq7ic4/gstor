package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chenq7an/gstor/common/block"
	"github.com/chenq7an/gstor/common/controller"
	"github.com/chenq7an/gstor/common/utils"
)

type smartReadTarget struct {
	DeviceLabel string
	Slot        string
	ReadCommand string
}

func resolveSmartReadTarget(target string) (smartReadTarget, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(target), "/dev/")
	if slot, err := block.ParseSlotID(trimmed); err == nil {
		return resolveSlotSmartReadTarget(slot)
	}

	collector, err := block.Devices()
	if err == nil {
		if mappedTarget, matched, err := resolveNamedSmartReadTarget(trimmed, collector.Collect(), resolveRaidSmartReadTarget); matched || err != nil {
			return mappedTarget, err
		}
	}

	return directSmartReadTarget(trimmed, ""), nil
}

func resolveSlotSmartReadTarget(slot block.SlotID) (smartReadTarget, error) {
	return resolveSlotSmartReadTargetWithResolvers(slot, resolveRaidSmartReadTarget, resolveMappedSmartReadTarget)
}

func resolveSlotSmartReadTargetWithResolvers(
	slot block.SlotID,
	raidResolver func(block.SlotID) (smartReadTarget, error),
	mappedResolver func(block.SlotID) (smartReadTarget, error),
) (smartReadTarget, error) {
	if slot.HasEnclosure() {
		return raidResolver(slot)
	}
	return mappedResolver(slot)
}

func resolveNamedSmartReadTarget(
	deviceName string,
	devices []block.Disk,
	raidResolver func(block.SlotID) (smartReadTarget, error),
) (smartReadTarget, bool, error) {
	var matches []block.Disk
	for _, device := range devices {
		if strings.TrimPrefix(device.Name, "/dev/") == deviceName {
			matches = append(matches, device)
		}
	}

	switch len(matches) {
	case 0:
		return smartReadTarget{}, false, nil
	case 1:
		match := matches[0]
		if slot, err := block.ParseSlotID(match.CES); err == nil && slot.HasEnclosure() {
			target, err := raidResolver(slot)
			if err != nil {
				return smartReadTarget{}, true, fmt.Errorf("failed to resolve raid smart access for %s via slot %s: %w", deviceName, slot.String(), err)
			}
			if target.Slot == "" {
				target.Slot = slot.String()
			}
			return target, true, nil
		}
		return directSmartReadTarget(deviceName, match.CES), true, nil
	default:
		return smartReadTarget{}, true, fmt.Errorf("device %s maps to multiple slots (%s); use c:e:s to read SMART", deviceName, joinKnownSlots(matches))
	}
}

func directSmartReadTarget(deviceName string, slot string) smartReadTarget {
	return smartReadTarget{
		DeviceLabel: deviceName,
		Slot:        normalizeSmartSlot(slot),
		ReadCommand: fmt.Sprintf(`smartctl -i -H -A /dev/%s`, deviceName),
	}
}

func normalizeSmartSlot(slot string) string {
	if slot == "" || strings.EqualFold(slot, "nil") {
		return ""
	}
	return slot
}

func joinKnownSlots(devices []block.Disk) string {
	slots := make([]string, 0, len(devices))
	seen := make(map[string]struct{}, len(devices))
	for _, device := range devices {
		slot := normalizeSmartSlot(device.CES)
		if slot == "" {
			continue
		}
		if _, ok := seen[slot]; ok {
			continue
		}
		seen[slot] = struct{}{}
		slots = append(slots, slot)
	}
	if len(slots) == 0 {
		return "unknown slots"
	}
	return strings.Join(slots, ", ")
}

func resolveMappedSmartReadTarget(slot block.SlotID) (smartReadTarget, error) {
	collector, err := block.Devices()
	if err != nil {
		return smartReadTarget{}, err
	}
	for _, device := range collector.Collect() {
		if device.CES != slot.String() {
			continue
		}
		if device.Name == "" || device.Name == "Nil" {
			return smartReadTarget{}, fmt.Errorf("slot %s is not mapped to a readable device node", slot.String())
		}
		name := strings.TrimPrefix(device.Name, "/dev/")
		return smartReadTarget{DeviceLabel: name, Slot: slot.String(), ReadCommand: fmt.Sprintf(`smartctl -i -H -A /dev/%s`, name)}, nil
	}
	return smartReadTarget{}, fmt.Errorf("slot %s not found", slot.String())
}

func resolveRaidSmartReadTarget(slot block.SlotID) (smartReadTarget, error) {
	if !slot.HasEnclosure() {
		return smartReadTarget{}, fmt.Errorf("raid smart requires c:e:s slot id")
	}

	ctrl := controller.Collect()
	switch ctrl.Tool {
	case controller.MegacliPath:
		deviceID := execTrimmed(fmt.Sprintf(`%s -Pdinfo -PhysDrv[%s:%s] -a%s | grep "Device Id" | awk -F: '{print $2}'`, ctrl.Tool, slot.EnclosureID, slot.SlotID, slot.ControllerID))
		busNumber := execTrimmed(fmt.Sprintf(`%s -adpgetpciinfo -a%s | grep "Bus Number" | awk '{print $NF}'`, ctrl.Tool, slot.ControllerID))
		return buildMegaraidReadTarget(slot, deviceID, busNumber)
	case controller.StorcliPath:
		deviceID := execTrimmed(fmt.Sprintf(`%s /c%s/e%s/s%s show all | grep "^%s:%s " | awk '{print $2}'`, ctrl.Tool, slot.ControllerID, slot.EnclosureID, slot.SlotID, slot.EnclosureID, slot.SlotID))
		busNumber := execTrimmed(fmt.Sprintf(`%s /c%s show all | grep "Bus Number" | awk '{print $NF}'`, ctrl.Tool, slot.ControllerID))
		return buildMegaraidReadTarget(slot, deviceID, busNumber)
	default:
		return smartReadTarget{}, fmt.Errorf("raid smart direct access is not supported for %s", ctrl.Tool)
	}
}

func buildMegaraidReadTarget(slot block.SlotID, deviceID string, busNumber string) (smartReadTarget, error) {
	if deviceID == "" {
		return smartReadTarget{}, fmt.Errorf("failed to resolve raid device id for slot %s", slot.String())
	}
	host, err := pciScsiHost(busNumber)
	if err != nil {
		return smartReadTarget{}, err
	}
	deviceLabel := fmt.Sprintf("megaraid:%s@/dev/bus/%s", deviceID, host)
	return smartReadTarget{DeviceLabel: deviceLabel, Slot: slot.String(), ReadCommand: fmt.Sprintf(`smartctl -i -H -A -d megaraid,%s /dev/bus/%s`, deviceID, host)}, nil
}

func pciScsiHost(busNumber string) (string, error) {
	candidates, err := pciBusCandidates(busNumber)
	if err != nil {
		return "", err
	}

	var lastErr error
	for _, bus := range candidates {
		entries, err := os.ReadDir(fmt.Sprintf(`/sys/bus/pci/devices/0000:%s:00.0/`, bus))
		if err != nil {
			lastErr = err
			continue
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "host") {
				return strings.TrimPrefix(entry.Name(), "host"), nil
			}
		}
		lastErr = fmt.Errorf("scsi host not found for pci bus %s", bus)
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("scsi host not found for pci bus %s", strings.TrimSpace(busNumber))
}

func pciBusCandidates(busNumber string) ([]string, error) {
	raw := strings.TrimSpace(busNumber)
	trimmed := strings.TrimPrefix(strings.ToLower(raw), "0x")
	if trimmed == "" {
		return nil, fmt.Errorf("empty pci bus number")
	}

	candidates := []string{zeroPadPCIBus(trimmed)}
	if strings.HasPrefix(strings.ToLower(raw), "0x") || !isDecimalString(trimmed) {
		return uniquePCIBuses(candidates), nil
	}

	decimalValue, err := strconv.Atoi(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid pci bus number %q: %w", busNumber, err)
	}
	candidates = append(candidates, fmt.Sprintf("%02x", decimalValue))
	return uniquePCIBuses(candidates), nil
}

func zeroPadPCIBus(value string) string {
	if len(value) == 1 {
		return "0" + value
	}
	return value
}

func isDecimalString(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func uniquePCIBuses(values []string) []string {
	unique := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func execTrimmed(cmd string) string {
	output, err := utils.ExecShell(cmd)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}
