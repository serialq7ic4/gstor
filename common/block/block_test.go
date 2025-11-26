package block

import (
	"testing"
)

// TestDiskCollectorInterface 测试 DiskCollector 接口的基本实现
// 注意：这是一个示例测试文件，实际的测试需要真实的硬件环境
func TestDiskCollectorInterface(t *testing.T) {
	// 这个测试主要验证接口定义是否正确
	// 实际的功能测试需要在有 RAID 控制器的环境中进行

	var collector DiskCollector

	// 测试 NVMe 收集器（不需要 RAID 控制器）
	nvmeCollector := &nvmeCollector{}
	collector = nvmeCollector

	// 验证接口方法存在
	_ = collector.Collect
	_ = collector.TurnOn
	_ = collector.TurnOff
}

// TestRaidToolAdapterRegistration 测试 RAID 工具适配器注册
func TestRaidToolAdapterRegistration(t *testing.T) {
	// 验证适配器已注册
	tools := GetSupportedRaidTools()

	if len(tools) == 0 {
		t.Error("No RAID tool adapters registered")
	}

	// 验证常见的工具已注册
	expectedTools := []string{
		"/opt/MegaRAID/MegaCli/MegaCli64",
		"/opt/MegaRAID/storcli/storcli64",
		"/usr/sbin/arcconf",
	}

	for _, expectedTool := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expectedTool {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %s not found in registered tools", expectedTool)
		}
	}
}

// TestDiskStruct 测试 Disk 结构体的字段
func TestDiskStruct(t *testing.T) {
	disk := Disk{
		Name:         "sda",
		CES:          "0:32:5",
		State:        "Online",
		MediaType:    "SSD",
		PDType:       "SAS",
		MediaError:   "0",
		PredictError: "0",
		Vendor:       "SEAGATE",
		Model:        "ST1000NM0011",
		Capacity:     "1 TB",
		SerialNumber: "Z1Z2Z3Z4",
	}

	if disk.Name == "" {
		t.Error("Disk name should not be empty")
	}

	if disk.CES == "" {
		t.Error("Disk CES should not be empty")
	}
}

// BenchmarkDiskCollect 基准测试（需要真实环境）
func BenchmarkDiskCollect(b *testing.B) {
	// 注意：这个基准测试需要在有 RAID 控制器的环境中运行
	// 在实际环境中取消注释以下代码

	/*
		collector, err := Devices()
		if err != nil {
			b.Skip("Skipping benchmark: no RAID controller available")
			return
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = collector.Collect()
		}
	*/

	b.Skip("Skipping benchmark: requires RAID controller hardware")
}
