package cmd

import (
	"fmt"
	"os"

	"github.com/chenq7an/gstor/common/controller"
	"github.com/chenq7an/gstor/common/utils"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var debugMode bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gstor",
	Short: "简短说明",
	Long:  `关于工具的长信息`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gstor.yaml)")

	// Debug flag - 全局可用
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "启用调试模式，显示详细的执行信息")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// IsDebugMode 返回当前是否处于 debug 模式
func IsDebugMode() bool {
	return debugMode
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// 初始化 debug 模式到 utils 包
	utils.SetDebugMode(debugMode)

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".gstor" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".gstor")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// Set default values from ToolMap
	for name, tool := range controller.ToolMap {
		viper.SetDefault("tools."+name, tool)
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
