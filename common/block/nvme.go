package block

import (
	"fmt"
	"strings"
)

func Nvme() []Disk {
	s := []Disk{}
	nvmeList := Bash(`lsblk | grep disk | grep nvme | awk '{print $1}'`)
	nvme := strings.Split(strings.Trim(nvmeList, "\n"), "\n")
	for _, v := range nvme {
		if v != "" {
			disk := Disk{CES: "Nil", State: "Online", MediaType: "SSD", PDType: "NVME", MediaError: "0", PredictError: "0"}
			disk.Name = v
			smartInfoSection := Bash(fmt.Sprintf(`smartctl /dev/%s -i`, disk.Name))
			smartInfo := strings.Split(strings.Trim(smartInfoSection, "\n"), "\n")
			for _, w := range smartInfo {
				switch {
				case strings.Contains(w, "Model Number"):
					disk.Vendor = strings.Split(strings.Trim(strings.Split(w, ":")[1], " "), " ")[0]
				case strings.Contains(w, "Serial Number"):
					disk.SerialNumber = strings.Trim(strings.Split(w, ":")[1], " ")
				case strings.Contains(w, "Total NVM Capacity"):
					disk.Capacity = strings.Split(strings.Trim(strings.Split(w, "[")[1], " "), "]")[0]
				case strings.Contains(w, "Size/Capacity:"):
					disk.Capacity = strings.Split(strings.Trim(strings.Split(w, "[")[1], " "), "]")[0]
				}
			}
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
