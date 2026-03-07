package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chenq7an/gstor/common/block"
	"github.com/chenq7an/gstor/common/utils"
	"github.com/spf13/cobra"
)

type smartResponse struct {
	RequestedTarget string             `json:"requested_target"`
	ResolvedDevice  string             `json:"resolved_device"`
	Slot            string             `json:"slot,omitempty"`
	Summary         block.SmartSummary `json:"summary"`
	RawOutput       string             `json:"raw_output,omitempty"`
}

var smartCmd = &cobra.Command{
	Use:   "smart <disk|c:e:s>",
	Short: "查看硬盘关键 SMART 信息",
	Long:  `支持盘符（如 sda、nvme0n1）或槽位（如 0:24:15）作为输入，输出关键 SMART 摘要`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		form, err := cmd.Flags().GetString("format")
		if err != nil {
			cobra.CheckErr(fmt.Errorf("failed to get format flag: %w", err))
		}
		if form != "" && form != "json" {
			cobra.CheckErr(fmt.Errorf("unsupported format %q, only json is supported", form))
		}

		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			cobra.CheckErr(fmt.Errorf("failed to get verbose flag: %w", err))
		}

		device, slot, err := resolveSmartDevice(args[0])
		if err != nil {
			cobra.CheckErr(err)
		}

		output, err := utils.ExecShell(fmt.Sprintf(`smartctl -i -H -A /dev/%s`, device))
		if err != nil {
			cobra.CheckErr(fmt.Errorf("failed to read smart info for %s: %w", device, err))
		}

		response := smartResponse{
			RequestedTarget: args[0],
			ResolvedDevice:  device,
			Slot:            slot,
			Summary:         block.ParseSmartSummary(device, output),
		}
		if verbose {
			response.RawOutput = output
		}

		if form == "json" {
			payload, err := json.Marshal(response)
			if err != nil {
				cobra.CheckErr(fmt.Errorf("failed to marshal smart response: %w", err))
			}
			fmt.Println(string(payload))
			return
		}

		printSmartSummary(response)
		if verbose {
			fmt.Println()
			fmt.Println("--- RAW SMART OUTPUT ---")
			fmt.Println(output)
		}
	},
}

func resolveSmartDevice(target string) (string, string, error) {
	if slot, err := block.ParseSlotID(target); err == nil {
		collector, err := block.Devices()
		if err != nil {
			return "", "", err
		}
		for _, device := range collector.Collect() {
			if device.CES != slot.String() {
				continue
			}
			if device.Name == "" || device.Name == "Nil" {
				return "", slot.String(), fmt.Errorf("slot %s is not mapped to a readable device node", slot.String())
			}
			return strings.TrimPrefix(device.Name, "/dev/"), slot.String(), nil
		}
		return "", slot.String(), fmt.Errorf("slot %s not found", slot.String())
	}

	return strings.TrimPrefix(strings.TrimSpace(target), "/dev/"), "", nil
}

func printSmartSummary(response smartResponse) {
	fields := []struct {
		label string
		value string
	}{
		{"Requested", response.RequestedTarget},
		{"Device", response.ResolvedDevice},
		{"Slot", response.Slot},
		{"Health", response.Summary.Health},
		{"Vendor", response.Summary.Vendor},
		{"Model", response.Summary.Model},
		{"Serial", response.Summary.SerialNumber},
		{"Capacity", response.Summary.Capacity},
		{"Temperature", formatTemperature(response.Summary.TemperatureC)},
		{"PowerOnHours", response.Summary.PowerOnHours},
		{"PowerCycles", response.Summary.PowerCycles},
		{"ReallocatedSectors", response.Summary.ReallocatedSectors},
		{"PendingSectors", response.Summary.PendingSectors},
		{"OfflineUncorrectable", response.Summary.OfflineUncorrectable},
		{"UDMACRCErrors", response.Summary.UDMACRCErrors},
		{"CriticalWarning", response.Summary.CriticalWarning},
		{"AvailableSpare", response.Summary.AvailableSpare},
		{"PercentageUsed", response.Summary.PercentageUsed},
		{"MediaErrors", response.Summary.MediaErrors},
		{"ErrorLogEntries", response.Summary.ErrorLogEntries},
	}

	for _, field := range fields {
		if field.value == "" {
			continue
		}
		fmt.Printf("%-20s %s\n", field.label+":", field.value)
	}
}

func formatTemperature(value string) string {
	if value == "" {
		return ""
	}
	if strings.ContainsAny(value, "Cc") {
		return value
	}
	return value + " C"
}

func init() {
	rootCmd.AddCommand(smartCmd)
	smartCmd.Flags().StringP("format", "f", "", "{json}")
	smartCmd.Flags().BoolP("verbose", "v", false, "显示原始 SMART 输出")
}
