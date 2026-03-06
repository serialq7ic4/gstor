package cmd

import (
	"fmt"

	"github.com/chenq7an/gstor/common/block"
	"github.com/chenq7an/gstor/common/controller"
	"github.com/chenq7an/gstor/common/utils"
	"github.com/spf13/cobra"
)

// mkraid0Cmd represents the mkraid0 command
var mkraid0Cmd = &cobra.Command{
	Use:   "mkraid0",
	Short: "对unconfigured硬盘制作raid0",
	Long:  `输入c:e:s信息,用于对该硬盘做raid0,目前仅适用于Megacli64控制器,具体c:e:s信息可以通过gstor list命令查看`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		slot, err := block.ParseSlotID(args[0])
		if err != nil {
			cobra.CheckErr(err)
		}

		if !slot.HasEnclosure() {
			cobra.CheckErr(fmt.Errorf("mkraid0 requires c:e:s slot id, got %q", args[0]))
		}

		c := controller.Collect()
		if c.Tool == controller.MegacliPath {
			cmd := fmt.Sprintf(`%s -CfgLdAdd -r0 [%s:%s] WB Direct -a%s`, c.Tool, slot.EnclosureID, slot.SlotID, slot.ControllerID)
			output, err := utils.ExecShell(cmd)
			if err != nil {
				cobra.CheckErr(fmt.Errorf("failed to create RAID0: %w, output: %s", err, output))
			}
			fmt.Println("RAID0 created successfully")
		} else {
			fmt.Printf("Not support yet, %s\n", c.Name)
		}
	},
}

func init() {
	rootCmd.AddCommand(mkraid0Cmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mkraid0Cmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mkraid0Cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
