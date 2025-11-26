package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// 版本信息变量，构建时通过 ldflags 注入
var (
	Version   = "dev"     // 版本号，构建时注入
	BuildTime = "unknown" // 构建时间，构建时注入
	GitCommit = "unknown" // Git 提交哈希，构建时注入
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "版本信息",
	Long:  `显示版本信息，包括版本号、构建时间和 Git 提交哈希`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
