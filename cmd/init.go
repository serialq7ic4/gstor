package cmd

import (
	"fmt"
	"os"

	"github.com/chenq7an/gstor/common/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化配置文件",
	Long:  `根据阵列卡型号生成配置文件，包含工具路径映射`,
	Run: func(cmd *cobra.Command, args []string) {
		// 获取当前控制器信息
		ctrl := controller.Collect()

		// 设置配置项
		viper.Set("controller.name", ctrl.Name)
		viper.Set("controller.tool", ctrl.Tool)
		viper.Set("controller.available", ctrl.Avail)

		// 写入配置文件
		configPath := getConfigPath()
		if err := viper.WriteConfigAs(configPath); err != nil {
			fmt.Printf("写入配置文件失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("配置文件已生成: %s\n", configPath)
	},
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gstor.yaml"
	}
	return home + "/.gstor.yaml"
}

func init() {
	rootCmd.AddCommand(initCmd)
}
