package cmd

import (
	"fmt"
	"os"

	"github.com/chenq7an/gstor/common/block"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "罗列出硬盘基本信息",
	Long:  `基于存储控制器展示硬盘诸如盘符、在控制器上的ces信息及硬盘状态等`,
	Run:   showBlock,
}

func showBlock(cmd *cobra.Command, args []string) {
	disk, err := block.Devices()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	devices := disk.Collect()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Disk", "SN", "Capacity", "Vendor", "MediaType", "Slot", "State", "MediaError", "PredictError"})
	t.AppendSeparator()
	for i := 0; i < len(devices); i++ {
		t.AppendRow(table.Row{devices[i].Name, devices[i].SerialNumber, devices[i].Capacity, devices[i].Vendor, devices[i].MediaType, devices[i].CES, devices[i].State, devices[i].MediaError, devices[i].PredictError})
	}
	t.SetStyle(table.StyleLight)
	t.SortBy([]table.SortBy{
		{Name: "Disk", Mode: table.Asc},
	})
	t.Render()
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
