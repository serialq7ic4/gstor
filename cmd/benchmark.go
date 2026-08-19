package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"

	"github.com/chenq7an/gstor/common/benchmark"
	"github.com/spf13/cobra"
)

var runBenchmark = benchmark.RunBenchmark

func newBenchmarkCommand() *cobra.Command {
	benchmarkCmd := &cobra.Command{
		Use:   "benchmark",
		Short: "硬盘基线性能测试",
		Long:  "执行安全裸盘校验后，通过 fio 对 eligible bare disks 采集基线性能数据",
	}
	benchmarkCmd.AddCommand(newBenchmarkRunCommand())
	return benchmarkCmd
}

func newBenchmarkRunCommand() *cobra.Command {
	var profile string
	var disks []string
	var outputPath string
	var format string
	var reportURL string

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "执行硬盘 benchmark",
		RunE: func(cmd *cobra.Command, args []string) error {
			if format == "" {
				format = "json"
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			result, err := runBenchmark(ctx, benchmark.RunOptions{
				ProfileName: profile,
				Disks:       disks,
				OutputPath:  outputPath,
				Format:      format,
				ReportURL:   reportURL,
				Stderr:      cmd.ErrOrStderr(),
			})
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetEscapeHTML(false)
			if err != nil {
				if len(result.Results) > 0 {
					if encodeErr := encoder.Encode(result); encodeErr != nil {
						return fmt.Errorf("failed to encode partial benchmark output: %w", encodeErr)
					}
				}
				return err
			}
			if err := encoder.Encode(result); err != nil {
				return fmt.Errorf("failed to encode benchmark output: %w", err)
			}
			return nil
		},
	}

	runCmd.Flags().StringVarP(&profile, "profile", "p", "default", "benchmark profile: default or short")
	runCmd.Flags().StringArrayVarP(&disks, "disk", "D", nil, "指定一个或多个裸盘，例如 sdb 或 /dev/sdb")
	runCmd.Flags().StringVarP(&outputPath, "output", "o", "", "额外写本地 JSON 文件")
	runCmd.Flags().StringVarP(&format, "format", "f", "json", "输出格式，目前仅支持 json")
	runCmd.Flags().StringVarP(&reportURL, "report-url", "r", "", "额外上报中心 API")
	return runCmd
}

func init() {
	rootCmd.AddCommand(newBenchmarkCommand())
}
