package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/chenq7an/gstor/common/block"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

type listDiskOutput struct {
	Name         string `json:"name"`
	CES          string `json:"ces"`
	State        string `json:"state"`
	MediaType    string `json:"mediatype"`
	PDType       string `json:"pdtype"`
	MediaError   string `json:"mediaerror"`
	PredictError string `json:"predicterror"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	Capacity     string `json:"capacity"`
	SerialNumber string `json:"serialnumber"`
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "罗列出硬盘基本信息",
	Long:  `基于存储控制器展示硬盘诸如盘符、在控制器上的ces信息及硬盘状态等`,
	Run: func(cmd *cobra.Command, args []string) {
		form, err := cmd.Flags().GetString("format")
		if err != nil {
			cobra.CheckErr(fmt.Errorf("failed to get format flag: %w", err))
		}
		if form != "" && form != "json" {
			cobra.CheckErr(fmt.Errorf("unsupported format %q, only json is supported", form))
		}
		_ = showBlock(form)
	},
}

func showBlock(form string) string {
	disk, err := block.Devices()
	if err != nil {
		cobra.CheckErr(err)
	}
	devices := disk.Collect()
	if form == "" {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Disk", "SN", "Capacity", "Vendor", "Model", "PDType", "MediaType", "Slot", "State", "MediaError", "PredictError"})
		t.AppendSeparator()
		for i := 0; i < len(devices); i++ {
			t.AppendRow(table.Row{devices[i].Name, devices[i].SerialNumber, devices[i].Capacity, devices[i].Vendor, devices[i].Model, devices[i].PDType, devices[i].MediaType, devices[i].CES, devices[i].State, devices[i].MediaError, devices[i].PredictError})
		}
		t.SetStyle(table.StyleLight)
		t.SortBy([]table.SortBy{{Name: "Disk", Mode: table.Asc}})
		t.Render()
		return "noformat"
	}

	var output []listDiskOutput
	for _, device := range devices {
		output = append(output, listDiskOutput{
			Name:         device.Name,
			CES:          device.CES,
			State:        device.State,
			MediaType:    device.MediaType,
			PDType:       device.PDType,
			MediaError:   device.MediaError,
			PredictError: device.PredictError,
			Vendor:       device.Vendor,
			Model:        device.Model,
			Capacity:     device.Capacity,
			SerialNumber: device.SerialNumber,
		})
	}

	r, err := json.Marshal(output)
	if err != nil {
		cobra.CheckErr(fmt.Errorf("failed to marshal JSON: %w", err))
	}
	fmt.Println(string(r))
	return string(r)
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("format", "f", "", "{json}")
}
