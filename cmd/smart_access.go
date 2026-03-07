package cmd

import (
	"fmt"
	"os"
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
		if raidTarget, err := resolveRaidSmartReadTarget(slot); err == nil {
			return raidTarget, nil
		}
		return resolveMappedSmartReadTarget(slot)
	}
	return smartReadTarget{DeviceLabel: trimmed, ReadCommand: fmt.Sprintf(`smartctl -i -H -A /dev/%s`, trimmed)}, nil
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
	bus := strings.TrimPrefix(strings.TrimSpace(busNumber), "0x")
	if bus == "" {
		return "", fmt.Errorf("empty pci bus number")
	}
	if len(bus) == 1 {
		bus = "0" + bus
	}
	entries, err := os.ReadDir(fmt.Sprintf(`/sys/bus/pci/devices/0000:%s:00.0/`, bus))
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "host") {
			return strings.TrimPrefix(entry.Name(), "host"), nil
		}
	}
	return "", fmt.Errorf("scsi host not found for pci bus %s", bus)
}

func execTrimmed(cmd string) string {
	output, err := utils.ExecShell(cmd)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}
