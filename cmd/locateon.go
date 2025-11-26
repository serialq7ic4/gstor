package cmd

import (
	"fmt"

	"github.com/chenq7an/gstor/common/block"
	"github.com/spf13/cobra"
)

// onCmd represents the on command
var onCmd = &cobra.Command{
	Use:   "on",
	Short: "打开硬盘状态灯",
	Long:  `通过硬盘 Slot 信息打开硬盘状态灯`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		disk, err := block.Devices()
		if err != nil {
			cobra.CheckErr(fmt.Errorf("failed to get devices: %w", err))
		}
		err = disk.TurnOn(args[0])
		if err != nil {
			cobra.CheckErr(fmt.Errorf("failed to turn on locate: %w", err))
		}
		fmt.Println("OK")
	},
}

func init() {
	locateCmd.AddCommand(onCmd)
}
