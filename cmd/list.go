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
	"gstor/common/block"
	"os"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "简短罗列出硬盘基本信息",
	Long:  `基于存储控制器展示硬盘诸如盘符、在控制器上的ces信息及硬盘状态等`,
	Run:   showBlock,
}

func showBlock(cmd *cobra.Command, args []string) {
	disk, err := block.Devices()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	devices := disk.Collect()
	fmt.Printf("%-6s %-22s %-10s %-10s %-10s %-10s %-20s %-10s %-12s\n", "Disk", "SN", "Capacity", "Vendor", "MediaType", "Slot", "State", "MediaError", "PredictError")
	for i := 0; i < len(devices); i++ {
		fmt.Printf("%-6s %-22s %-10s %-10s %-10s %-10s %-20s %-10s %-12s\n", devices[i].Name, devices[i].SerialNumber, devices[i].Capacity, devices[i].Vendor, devices[i].MediaType, devices[i].CES, devices[i].State, devices[i].MediaError, devices[i].PredictError)
	}
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
