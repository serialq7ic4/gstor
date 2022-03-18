package cmd

import (
	"fmt"
	"strings"

	"github.com/chenq7an/gstor/common/controller"
	"github.com/spf13/cobra"
)

// mkraid0Cmd represents the mkraid0 command
var mkraid0Cmd = &cobra.Command{
	Use:   "mkraid0",
	Short: "对unconfigured硬盘制作raid0",
	Long:  `输入c:e:s信息,用于对该硬盘做raid0,目前仅适用于Megacli64控制器,具体c:e:s信息可以通过gstor list命令查看`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := controller.Collect()
		cid := strings.Split(args[0], ":")[0]
		eid := strings.Split(args[0], ":")[1]
		sid := strings.Split(args[0], ":")[2]
		if c.Tool == "/opt/MegaRAID/MegaCli/MegaCli64" {
			bash(fmt.Sprintf(`%s -cfgforeign -clear -aALL`, c.Tool))
			bash(fmt.Sprintf(`%s -CfgLdAdd -r0 [%s:%s] WB Direct -a%s`, c.Tool, eid, sid, cid))
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
