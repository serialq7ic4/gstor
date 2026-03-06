package block

import (
	"fmt"
	"strings"
)

type SlotID struct {
	ControllerID string
	EnclosureID  string
	SlotID       string
}

func ParseSlotID(value string) (SlotID, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	switch len(parts) {
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return SlotID{}, fmt.Errorf("invalid slot id %q, expected c:s", value)
		}
		return SlotID{
			ControllerID: parts[0],
			SlotID:       parts[1],
		}, nil
	case 3:
		if parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return SlotID{}, fmt.Errorf("invalid slot id %q, expected c:e:s", value)
		}
		return SlotID{
			ControllerID: parts[0],
			EnclosureID:  parts[1],
			SlotID:       parts[2],
		}, nil
	default:
		return SlotID{}, fmt.Errorf("invalid slot id %q, expected c:s or c:e:s", value)
	}
}

func (slot SlotID) HasEnclosure() bool {
	return slot.EnclosureID != ""
}

func (slot SlotID) String() string {
	if slot.HasEnclosure() {
		return fmt.Sprintf("%s:%s:%s", slot.ControllerID, slot.EnclosureID, slot.SlotID)
	}
	return fmt.Sprintf("%s:%s", slot.ControllerID, slot.SlotID)
}
