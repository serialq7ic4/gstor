package cmd

import (
	"fmt"

	"github.com/chenq7an/gstor/common/block"
	"github.com/spf13/cobra"
)

// offCmd represents the off command
var offCmd = &cobra.Command{
	Use:   "off",
	Short: "关闭硬盘状态灯",
	Long:  `通过硬盘 Slot 信息关闭硬盘状态灯`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		disk, err := block.Devices()
		if err != nil {
			cobra.CheckErr(fmt.Errorf("failed to get devices: %w", err))
		}
		err = disk.TurnOff(args[0])
		if err != nil {
			cobra.CheckErr(fmt.Errorf("failed to turn off locate: %w", err))
		}
		fmt.Println("OK")
	},
}

func init() {
	locateCmd.AddCommand(offCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// offCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// offCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
