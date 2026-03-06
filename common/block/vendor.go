package block

import "strings"

func NormalizeVendor(vendor string) string {
	vendor = strings.TrimSpace(vendor)
	upper := strings.ToUpper(vendor)

	switch {
	case strings.HasPrefix(upper, "ST"):
		return "SEAGATE"
	case strings.HasPrefix(upper, "HU"):
		return "HGST"
	case strings.HasPrefix(upper, "MICRON"):
		return "MICRON"
	default:
		return vendor
	}
}
