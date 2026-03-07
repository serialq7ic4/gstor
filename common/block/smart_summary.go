package block

import "strings"

type SmartSummary struct {
	Device               string `json:"device"`
	Health               string `json:"health,omitempty"`
	Vendor               string `json:"vendor,omitempty"`
	Model                string `json:"model,omitempty"`
	SerialNumber         string `json:"serialnumber,omitempty"`
	Capacity             string `json:"capacity,omitempty"`
	TemperatureC         string `json:"temperature_c,omitempty"`
	PowerOnHours         string `json:"power_on_hours,omitempty"`
	PowerCycles          string `json:"power_cycles,omitempty"`
	ReallocatedSectors   string `json:"reallocated_sectors,omitempty"`
	PendingSectors       string `json:"pending_sectors,omitempty"`
	OfflineUncorrectable string `json:"offline_uncorrectable,omitempty"`
	UDMACRCErrors        string `json:"udma_crc_errors,omitempty"`
	CriticalWarning      string `json:"critical_warning,omitempty"`
	AvailableSpare       string `json:"available_spare,omitempty"`
	PercentageUsed       string `json:"percentage_used,omitempty"`
	MediaErrors          string `json:"media_errors,omitempty"`
	ErrorLogEntries      string `json:"error_log_entries,omitempty"`
}

func ParseSmartSummary(device string, output string) SmartSummary {
	summary := SmartSummary{Device: device}
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		switch {
		case strings.HasPrefix(line, "Device Model:"):
			summary.Model = strings.TrimSpace(strings.TrimPrefix(line, "Device Model:"))
		case strings.HasPrefix(line, "Model Number:"):
			summary.Model = strings.TrimSpace(strings.TrimPrefix(line, "Model Number:"))
		case strings.HasPrefix(line, "Product:") && summary.Model == "":
			summary.Model = strings.TrimSpace(strings.TrimPrefix(line, "Product:"))
		case strings.HasPrefix(line, "Vendor:"):
			summary.Vendor = strings.TrimSpace(strings.TrimPrefix(line, "Vendor:"))
		case strings.HasPrefix(line, "Serial Number:"):
			summary.SerialNumber = strings.TrimSpace(strings.TrimPrefix(line, "Serial Number:"))
		case strings.HasPrefix(line, "User Capacity:"):
			summary.Capacity = extractBracketValue(line, "User Capacity:")
		case strings.HasPrefix(line, "Total NVM Capacity:"):
			summary.Capacity = extractBracketValue(line, "Total NVM Capacity:")
		case strings.HasPrefix(line, "SMART overall-health self-assessment test result:"):
			summary.Health = strings.TrimSpace(strings.TrimPrefix(line, "SMART overall-health self-assessment test result:"))
		case strings.HasPrefix(line, "SMART Health Status:"):
			summary.Health = strings.TrimSpace(strings.TrimPrefix(line, "SMART Health Status:"))
		case strings.HasPrefix(line, "Critical Warning:"):
			summary.CriticalWarning = strings.TrimSpace(strings.TrimPrefix(line, "Critical Warning:"))
		case strings.HasPrefix(line, "Temperature:"):
			summary.TemperatureC = firstField(strings.TrimSpace(strings.TrimPrefix(line, "Temperature:")))
		case strings.HasPrefix(line, "Available Spare:"):
			summary.AvailableSpare = strings.TrimSpace(strings.TrimPrefix(line, "Available Spare:"))
		case strings.HasPrefix(line, "Percentage Used:"):
			summary.PercentageUsed = strings.TrimSpace(strings.TrimPrefix(line, "Percentage Used:"))
		case strings.HasPrefix(line, "Power On Hours:"):
			summary.PowerOnHours = firstField(strings.TrimSpace(strings.TrimPrefix(line, "Power On Hours:")))
		case strings.HasPrefix(line, "Power Cycles:"):
			summary.PowerCycles = firstField(strings.TrimSpace(strings.TrimPrefix(line, "Power Cycles:")))
		case strings.HasPrefix(line, "Media and Data Integrity Errors:"):
			summary.MediaErrors = firstField(strings.TrimSpace(strings.TrimPrefix(line, "Media and Data Integrity Errors:")))
		case strings.HasPrefix(line, "Error Information Log Entries:"):
			summary.ErrorLogEntries = firstField(strings.TrimSpace(strings.TrimPrefix(line, "Error Information Log Entries:")))
		default:
			parseSmartAttribute(line, &summary)
		}
	}
	return summary
}

func parseSmartAttribute(line string, summary *SmartSummary) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return
	}

	attributeName := fields[1]
	rawValue := smartAttributeRawValue(fields)
	switch attributeName {
	case "Power_On_Hours", "Power_On_Hours_and_Msec":
		summary.PowerOnHours = rawValue
	case "Power_Cycle_Count":
		summary.PowerCycles = rawValue
	case "Temperature_Celsius", "Airflow_Temperature_Cel", "Temperature_Internal":
		summary.TemperatureC = rawValue
	case "Reallocated_Sector_Ct":
		summary.ReallocatedSectors = rawValue
	case "Current_Pending_Sector":
		summary.PendingSectors = rawValue
	case "Offline_Uncorrectable":
		summary.OfflineUncorrectable = rawValue
	case "UDMA_CRC_Error_Count":
		summary.UDMACRCErrors = rawValue
	}
}

func smartAttributeRawValue(fields []string) string {
	if len(fields) >= 9 {
		return strings.Trim(fields[8], "(),")
	}
	return strings.Trim(fields[len(fields)-1], "(),")
}

func extractBracketValue(line string, prefix string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if strings.Contains(trimmed, "[") && strings.Contains(trimmed, "]") {
		parts := strings.Split(trimmed, "[")
		if len(parts) > 1 {
			value := strings.Split(parts[1], "]")
			if len(value) > 0 {
				return strings.TrimSpace(value[0])
			}
		}
	}
	return trimmed
}

func firstField(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
