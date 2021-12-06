package cmd

import (
	"github.com/spf13/cobra"
)

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:   "locate",
	Short: "用于点亮/熄灭硬盘灯",
	Long:  `通过提供controller id、enclosive id及slot id信息来点亮/熄灭硬盘灯的命令，参数形如 0:1:2`,
	Run:   locate,
}

func locate(cmd *cobra.Command, args []string) {
	cmd.Help()
}

func init() {
	rootCmd.AddCommand(locateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// locateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// locateCmd.Flags().StringP("slot-info", "s", "", "硬盘slot信息")
}
