package block

import (
	"fmt"

	"github.com/chenq7an/gstor/common/controller"
	"github.com/chenq7an/gstor/common/utils"
	"github.com/spf13/viper"
)

func init() {
	viper.AutomaticEnv()
	// 注册所有 RAID 工具适配器
	registerRaidToolAdapters()
}

type Disk struct {
	Name         string `json:"name"`
	CES          string `json:"ces"`
	State        string `json:"state"`
	MediaType    string `json:"mediatype"`
	PDType       string `json:"pdtype"`
	MediaError   string `json:"mediaerror"`
	PredictError string `json:"predicterror"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	Capacity     string `json:"capcity"`
	SerialNumber string `json:"serialnumber"`
}

type DiskCollector interface {
	Collect() []Disk
	TurnOn(slot string) error
	TurnOff(slot string) error
}

// RaidToolAdapter 定义 RAID 工具适配器接口
type RaidToolAdapter interface {
	CreateCollector() DiskCollector
	SupportedTool() string
}

// 全局 RAID 工具适配器注册表
var raidToolAdapters = make(map[string]RaidToolAdapter)

// RegisterRaidToolAdapter 注册 RAID 工具适配器
func RegisterRaidToolAdapter(toolPath string, adapter RaidToolAdapter) {
	raidToolAdapters[toolPath] = adapter
}

// GetSupportedRaidTools 获取所有支持的 RAID 工具路径
func GetSupportedRaidTools() []string {
	tools := make([]string, 0, len(raidToolAdapters))
	for tool := range raidToolAdapters {
		tools = append(tools, tool)
	}
	return tools
}

// RAID 工具适配器实现
type megacliAdapter struct{}

func (a *megacliAdapter) CreateCollector() DiskCollector {
	return &megacliCollector{}
}

func (a *megacliAdapter) SupportedTool() string {
	return controller.MegacliPath
}

type storcliAdapter struct{}

func (a *storcliAdapter) CreateCollector() DiskCollector {
	return &storcliCollector{}
}

func (a *storcliAdapter) SupportedTool() string {
	return controller.StorcliPath
}

type arcconfAdapter struct{}

func (a *arcconfAdapter) CreateCollector() DiskCollector {
	return &arcconfCollector{}
}

func (a *arcconfAdapter) SupportedTool() string {
	return controller.ArcconfPath
}

// NVMe 适配器实现
type nvmeAdapter struct{}

func (a *nvmeAdapter) CreateCollector() DiskCollector {
	return &nvmeCollector{}
}

func (a *nvmeAdapter) SupportedTool() string {
	return "nvme"
}

// registerRaidToolAdapters 注册所有 RAID 工具适配器
func registerRaidToolAdapters() {
	RegisterRaidToolAdapter(controller.MegacliPath, &megacliAdapter{})
	RegisterRaidToolAdapter(controller.StorcliPath, &storcliAdapter{})
	RegisterRaidToolAdapter(controller.ArcconfPath, &arcconfAdapter{})
}

// Bash 执行 shell 命令并返回输出
// 错误时返回空字符串（为了向后兼容）
// 新代码应该使用 utils.ExecShell() 来获取错误信息
// Deprecated: 使用 utils.ExecShell() 替代，以便正确处理错误
func Bash(cmd string) string {
	return utils.ExecShellSafe(cmd)
}

func Devices() (DiskCollector, error) {
	// 优先使用配置文件中的工具路径
	raidTool := viper.GetString("controller.tool")
	if raidTool == "" {
		c := controller.Collect()
		raidTool = c.Tool
	}

	// 如果没有检测到 RAID 工具，只返回 NVMe 收集器
	if raidTool == controller.UnknownTool || raidTool == "" {
		return &nvmeCollector{}, nil
	}

	// 从注册表中查找对应的 RAID 工具适配器
	toolAdapter, exists := raidToolAdapters[raidTool]
	if !exists {
		return nil, fmt.Errorf("unsupported raid tool: %s, supported tools: %v",
			raidTool, GetSupportedRaidTools())
	}

	return &combinedCollector{
		raidCollector: toolAdapter.CreateCollector(),
		nvmeCollector: &nvmeCollector{},
	}, nil
}

// combinedCollector 组合 RAID 和 NVMe 收集器
type combinedCollector struct {
	raidCollector DiskCollector
	nvmeCollector DiskCollector
}

func (c *combinedCollector) Collect() []Disk {
	var allDisks []Disk

	// 收集 RAID 硬盘
	raidDisks := c.raidCollector.Collect()
	allDisks = append(allDisks, raidDisks...)

	// 收集 NVMe 硬盘
	nvmeDisks := c.nvmeCollector.Collect()
	if len(nvmeDisks) > 0 {
		allDisks = append(allDisks, nvmeDisks...)
	}

	return allDisks
}

func (c *combinedCollector) TurnOn(slot string) error {
	// 首先尝试 RAID 收集器
	if err := c.raidCollector.TurnOn(slot); err == nil {
		return nil
	}

	// 如果 RAID 收集器失败，尝试 NVMe 收集器
	return c.nvmeCollector.TurnOn(slot)
}

func (c *combinedCollector) TurnOff(slot string) error {
	// 首先尝试 RAID 收集器
	if err := c.raidCollector.TurnOff(slot); err == nil {
		return nil
	}

	// 如果 RAID 收集器失败，尝试 NVMe 收集器
	return c.nvmeCollector.TurnOff(slot)
}
