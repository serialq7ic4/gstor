package block

import (
	"fmt"
	"path/filepath"
	"strings"
)

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
	// 执行lspci命令
	lspciInfoSection := Bash(fmt.Sprintf(`lspci -vvs %s | grep "Physical Slot" | awk '{print $NF}'`, pciID))
	phyid := strings.Trim(lspciInfoSection, "\n")
	return phyid, nil
}

func Nvme() []Disk {
	s := []Disk{}
	nvmeList := Bash(`lsblk | grep disk | grep nvme | awk '{print $1}'`)
	nvme := strings.Split(strings.Trim(nvmeList, "\n"), "\n")
	for _, v := range nvme {
		if v != "" {
			disk := Disk{CES: "Nil", State: "Direct", MediaType: "SSD", PDType: "NVME", MediaError: "0", PredictError: "0"}
			disk.Name = v
			smartInfoSection := Bash(fmt.Sprintf(`smartctl /dev/%s -i`, disk.Name))
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
					// disk.Vendor = strings.Split(strings.Trim(strings.Split(w, ":")[1], " "), " ")[0]
					// disk.Model = strings.Split(strings.Trim(strings.Split(w, ":")[1], " "), " ")[0]
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
				fmt.Println("Error evaluating symlink:", err)
			}
			pciID := extractPCIID(targetPath)
			physicalSlot, err := getPhysicalSlot(pciID)
			if err != nil {
				fmt.Println("Error getting Physical Slot:", err)
			}
			disk.CES = physicalSlot
			smartSection := Bash(fmt.Sprintf(`smartctl /dev/%s -A`, disk.Name))
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
