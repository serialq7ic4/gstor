package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/chenq7an/gstor/common/benchmark"
	"github.com/chenq7an/gstor/common/controller"

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

	report := benchmark.CheckRequirements(benchmark.SystemProbe{})
	nvmeCheck, nvmeCheckErr := benchmark.CheckNVMeSmartctlCompatibility(context.Background(), benchmark.SystemRunner{})
	benchmarkReady := report.Ready
	if nvmeCheckErr != nil {
		benchmarkReady = false
		fmt.Printf("Benchmark NVMe 检查: false (%v)\n", nvmeCheckErr)
	} else if !nvmeCheck.HasNVMe {
		fmt.Println("Benchmark NVMe 检查: 未发现 NVMe 硬盘")
	} else if !nvmeCheck.Ready {
		benchmarkReady = false
		fmt.Printf("Benchmark NVMe 检查: false (发现 %s，%s)\n", strings.Join(nvmeCheck.Devices, ","), nvmeCheck.Warning)
	} else {
		fmt.Printf("Benchmark NVMe 检查: true (发现 %s，smartmontools %s)\n", strings.Join(nvmeCheck.Devices, ","), nvmeCheck.Version)
	}
	fmt.Printf("Benchmark 条件满足: %t\nRoot 权限: %t\n", benchmarkReady, report.Root)
	for _, dep := range report.Dependencies {
		fmt.Printf("Benchmark 依赖 %-8s: %t", dep.Name, dep.Available)
		if dep.Path != "" {
			fmt.Printf(" (%s)", dep.Path)
		}
		fmt.Println()
	}
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
