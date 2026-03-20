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
	if summary.TemperatureC != "34" {
		t.Fatalf("unexpected temperature: %+v", summary)
	}
}

func TestParseSmartSummaryAnnotatedTemperature(t *testing.T) {
	output := `
190 Airflow_Temperature_Cel 072 058 045 Old_age Always - 28 (Min/Max 24/32)
194 Temperature_Celsius     040   052   000    Old_age   Always       -       40 (0 18 0 0 0)
`
	summary := ParseSmartSummary("sdb", output)
	if summary.TemperatureC != "40" {
		t.Fatalf("expected annotated temperature to parse as 40, got %+v", summary)
	}
}

func TestParseSmartSummaryMegaraidAttributeTable(t *testing.T) {
	output := `
Device Model:     HGST HUS728T8TALE6L4
Serial Number:    VY1VWW7M
SMART overall-health self-assessment test result: PASSED
  5 Reallocated_Sector_Ct   0x0033   100   100   005    Pre-fail  Always       -       0
  9 Power_On_Hours          0x0012   100   100   000    Old_age   Always       -       6397
 12 Power_Cycle_Count       0x0032   100   100   000    Old_age   Always       -       33
194 Temperature_Celsius     0x0002   214   214   000    Old_age   Always       -       28 (Min/Max 25/42)
197 Current_Pending_Sector  0x0022   100   100   000    Old_age   Always       -       0
198 Offline_Uncorrectable   0x0008   100   100   000    Old_age   Offline      -       0
199 UDMA_CRC_Error_Count    0x000a   200   200   000    Old_age   Always       -       0
`
	summary := ParseSmartSummary("megaraid:6@/dev/bus/0", output)
	if summary.PowerOnHours != "6397" || summary.PowerCycles != "33" {
		t.Fatalf("unexpected megaraid counters: %+v", summary)
	}
	if summary.TemperatureC != "28" {
		t.Fatalf("unexpected megaraid temperature: %+v", summary)
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
