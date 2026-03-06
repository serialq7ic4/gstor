package block

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenq7an/gstor/common/utils"
)

// nvmeCollector 实现 DiskCollector 接口
type nvmeCollector struct{}

func (n *nvmeCollector) Collect() []Disk {
	return Nvme()
}

func (n *nvmeCollector) TurnOn(slot string) error {
	// NVMe 硬盘通常没有 RAID 卡控制的点灯功能
	return fmt.Errorf("nvme disk %s does not support locate function", slot)
}

func (n *nvmeCollector) TurnOff(slot string) error {
	// NVMe 硬盘通常没有 RAID 卡控制的点灯功能
	return fmt.Errorf("nvme disk %s does not support locate function", slot)
}

// 从路径中提取最后一个PCI设备ID
func extractPCIID(path string) string {
	// 示例路径：../devices/pci0000:10/0000:10:01.2/0000:12:00.0/nvme/nvme0/nvme0n1
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- { // 从后往前遍历
		if strings.HasPrefix(parts[i], "0000:") {
			return parts[i] // 返回最后一个符合条件的部分
		}
	}
	return ""
}

// 获取指定PCI设备的Physical Slot信息
func getPhysicalSlot(pciID string) (string, error) {
	output, err := utils.ExecShell(fmt.Sprintf(`lspci -vvs %s | grep "Physical Slot" | awk '{print $NF}'`, pciID))
	if err != nil {
		return "", err
	}
	return strings.Trim(output, "\n"), nil
}

func Nvme() []Disk {
	s := []Disk{}
	nvmeList, err := utils.ExecShell(`lsblk | grep disk | grep nvme | awk '{print $1}'`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to list NVMe disks: %v\n", err)
		return s
	}

	nvme := strings.Split(strings.Trim(nvmeList, "\n"), "\n")
	for _, v := range nvme {
		if v != "" {
			disk := Disk{CES: "Nil", State: "Direct", MediaType: "SSD", PDType: "NVME", MediaError: "0", PredictError: "0"}
			disk.Name = v
			smartInfoSection, err := utils.ExecShell(fmt.Sprintf(`smartctl /dev/%s -i`, disk.Name))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to read smart info for %s: %v\n", disk.Name, err)
				smartInfoSection = ""
			}
			smartInfo := strings.Split(strings.Trim(smartInfoSection, "\n"), "\n")
			for _, w := range smartInfo {
				switch {
				case strings.Contains(w, "Model Number"):
					parts := strings.Split(strings.Trim(strings.Split(w, ":")[1], " "), " ")
					disk.Vendor = parts[0]
					if len(parts) > 1 {
						disk.Model = parts[1]
					} else {
						disk.Model = disk.Vendor
					}
				case strings.Contains(w, "Serial Number"):
					disk.SerialNumber = strings.Trim(strings.Split(w, ":")[1], " ")
				case strings.Contains(w, "Total NVM Capacity"):
					disk.Capacity = strings.Split(strings.Trim(strings.Split(w, "[")[1], " "), "]")[0]
				case strings.Contains(w, "Size/Capacity:"):
					disk.Capacity = strings.Split(strings.Trim(strings.Split(w, "[")[1], " "), "]")[0]
				}
			}
			linkPath := filepath.Join("/sys/block", disk.Name)
			targetPath, err := filepath.EvalSymlinks(linkPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to evaluate symlink %s: %v\n", linkPath, err)
				disk.CES = "Nil"
			} else {
				pciID := extractPCIID(targetPath)
				physicalSlot, err := getPhysicalSlot(pciID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to get Physical Slot for %s: %v\n", pciID, err)
					disk.CES = "Nil"
				} else {
					disk.CES = physicalSlot
				}
			}
			smartSection, err := utils.ExecShell(fmt.Sprintf(`smartctl /dev/%s -A`, disk.Name))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to read smart attributes for %s: %v\n", disk.Name, err)
				smartSection = ""
			}
			smart := strings.Split(strings.Trim(smartSection, "\n"), "\n")
			for _, x := range smart {
				switch {
				case strings.Contains(x, "Media and Data Integrity Errors"):
					disk.MediaError = strings.Trim(strings.Split(x, ":")[1], " ")
				case strings.Contains(x, "Error Information Log Entries"):
					disk.PredictError = strings.Trim(strings.Split(x, ":")[1], " ")
				}
			}
			s = append(s, disk)
		}
	}
	return s
}
