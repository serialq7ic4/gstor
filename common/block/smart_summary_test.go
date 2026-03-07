package block

import "testing"

func TestParseSmartSummarySATA(t *testing.T) {
	output := `
Device Model:     ST1000NM0011
Serial Number:    Z1ABCDEF
User Capacity:    1,000,204,886,016 bytes [1.00 TB]
SMART overall-health self-assessment test result: PASSED
  5 Reallocated_Sector_Ct   100   100   036    Pre-fail  Always       -       2
  9 Power_On_Hours          095   095   000    Old_age   Always       -       1234
194 Temperature_Celsius     066   052   000    Old_age   Always       -       34
197 Current_Pending_Sector  100   100   000    Old_age   Always       -       1
198 Offline_Uncorrectable   100   100   000    Old_age   Offline      -       0
199 UDMA_CRC_Error_Count    200   200   000    Old_age   Always       -       7
`
	summary := ParseSmartSummary("sda", output)
	if summary.Device != "sda" || summary.Health != "PASSED" {
		t.Fatalf("unexpected basic fields: %+v", summary)
	}
	if summary.ReallocatedSectors != "2" || summary.PendingSectors != "1" || summary.UDMACRCErrors != "7" {
		t.Fatalf("unexpected sata attributes: %+v", summary)
	}
}

func TestParseSmartSummaryNVMe(t *testing.T) {
	output := `
Model Number:                       MICRON_7400_MTFDKBA1T9TDQ
Serial Number:                      1234ABC
Total NVM Capacity:                 1,920,383,410,176 [1.92 TB]
SMART Health Status:                OK
Critical Warning:                   0x00
Temperature:                        39 Celsius
Available Spare:                    100%
Percentage Used:                    2%
Power Cycles:                       31
Power On Hours:                     456
Media and Data Integrity Errors:    0
Error Information Log Entries:      12
`
	summary := ParseSmartSummary("nvme0n1", output)
	if summary.Health != "OK" || summary.TemperatureC != "39" {
		t.Fatalf("unexpected nvme fields: %+v", summary)
	}
	if summary.AvailableSpare != "100%" || summary.PercentageUsed != "2%" || summary.ErrorLogEntries != "12" {
		t.Fatalf("unexpected nvme metrics: %+v", summary)
	}
}
