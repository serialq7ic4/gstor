/*
Copyright © 2021 chenqian

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"gstor/common/controller"

	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "获取存储相关设备概览说明",
	Long:  `展示存储控制器、控制器命令行工具选择、控制器是否安装、总硬盘数及硬盘在线数等概览信息`,
	Run:   showController,
}

func showController(cmd *cobra.Command, args []string) {
	ctrl := controller.Collect()
	fmt.Printf("存储控制器: %s\n命令行工具: %s\n工具已安装: %t\n", ctrl.Name, ctrl.Tool, ctrl.Avail)
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// checkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// checkCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
