package controller

import (
	"testing"
)

// TestPathExists 测试路径存在性检查
func TestPathExists(t *testing.T) {
	// 测试存在的路径
	if !PathExists("/") {
		t.Error("Root path should exist")
	}

	// 测试不存在的路径
	if PathExists("/nonexistent/path/that/should/not/exist") {
		t.Error("Non-existent path should return false")
	}
}

// TestChooseTool 测试工具选择逻辑
func TestChooseTool(t *testing.T) {
	tests := []struct {
		name       string
		controller string
		expected   string
	}{
		{
			name:       "MegaRAID SAS 2208",
			controller: "LSI Logic / Symbios Logic MegaRAID SAS 2208",
			expected:   MegacliPath,
		},
		{
			name:       "SAS3008",
			controller: "Broadcom / LSI SAS3008 PCI-Express Fusion-MPT SAS-3",
			expected:   StorcliPath,
		},
		{
			name:       "Adaptec Series 8",
			controller: "Adaptec Series 8 12G SAS/PCIe 3",
			expected:   ArcconfPath,
		},
		{
			name:       "Unknown controller",
			controller: "Unknown Controller",
			expected:   UnknownTool,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ChooseTool(tt.controller)
			if result != tt.expected {
				t.Errorf("ChooseTool(%q) = %q, want %q", tt.controller, result, tt.expected)
			}
		})
	}
}

// TestToolMap 测试工具映射表
func TestToolMap(t *testing.T) {
	if len(ToolMap) == 0 {
		t.Error("ToolMap should not be empty")
	}

	// 验证常见控制器都在映射表中
	commonControllers := []string{
		"LSI Logic / Symbios Logic MegaRAID SAS 2208",
		"Broadcom / LSI SAS3008 PCI-Express Fusion-MPT SAS-3",
		"Adaptec Series 8 12G SAS/PCIe 3",
	}

	for _, controller := range commonControllers {
		if _, exists := ToolMap[controller]; !exists {
			t.Errorf("Controller %q not found in ToolMap", controller)
		}
	}
}

// TestControllerStruct 测试 Controller 结构体
func TestControllerStruct(t *testing.T) {
	ctrl := Controller{
		Name:  "Test Controller",
		Num:   1,
		Tool:  MegacliPath,
		Avail: true,
	}

	if ctrl.Name == "" {
		t.Error("Controller name should not be empty")
	}

	if ctrl.Num < 0 {
		t.Error("Controller number should be non-negative")
	}
}
