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
			fmt.Println(err)
		}
		err = disk.TurnOn(args[0])
		fmt.Println(err)
	},
}

func init() {
	locateCmd.AddCommand(onCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// onCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// onCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
