package block

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/chenq7an/gstor/common/controller"
	"github.com/spf13/cast"
)

type megacliCollector struct{}

func megacli(id string, results chan<- Disk, wg *sync.WaitGroup) {
	var _deviceId string
	var _wwn string

	tool := controller.MegacliPath
	defer wg.Done()

	// fmt.Printf("Device %s collecting\n", id)

	// 解析 ID，格式：c:e:s
	parts := strings.Split(id, ":")
	if len(parts) != 3 {
		fmt.Printf("Invalid device ID format: %s, expected format: c:e:s\n", id)
		return
	}

	cid := parts[0]
	eid := parts[1]
	sid := parts[2]

	disk := Disk{CES: id, MediaError: "0", PredictError: "0"}
	// 从阵列卡 Pdinfo 中抓取的信息
	megacliInfo := Bash(fmt.Sprintf(`%s -Pdinfo -PhysDrv[%s:%s] -a%s | egrep "Device Id:|WWN:|Firmware state:|Media Type:|Media Error Count:|Other Error Count:|Predictive Failure Count:|PD Type:"`, tool, eid, sid, cid))

	pdInfo := strings.Split(strings.Trim(megacliInfo, "\n"), "\n")

	for _, v := range pdInfo {
		switch {
		case strings.Contains(v, "Device Id"):
			_deviceId = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "WWN"):
			_wwn = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "Firmware state"):
			disk.State = strings.Trim(strings.Split(strings.Split(v, ":")[1], ",")[0], " ")
		case strings.Contains(v, "Media Type"):
			var mediaType string
			tmpResult := strings.Split(strings.Trim(strings.Split(v, ":")[1], " "), " ")
			for _, t := range tmpResult {
				mediaType += string(t[0])
			}
			disk.MediaType = mediaType
		case strings.Contains(v, "PD Type"):
			disk.PDType = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "Media Error Count"):
			merr := strings.Trim(strings.Split(v, ":")[1], " ")
			disk.MediaError = cast.ToString(cast.ToInt(merr) + cast.ToInt(disk.MediaError))
		case strings.Contains(v, "Other Error Count"):
			disk.PredictError = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "Predictive Failure Count"):
			perr := strings.Trim(strings.Split(v, ":")[1], " ")
			disk.PredictError = cast.ToString(cast.ToInt(perr) + cast.ToInt(disk.PredictError))
		}
	}

	// 从 SMART 中抓取的信息
	var scsiBusNumber string
	adapterInfo := Bash(fmt.Sprintf(`%s -adpgetpciinfo -a%s | grep "Bus Number" | awk '{print $NF}'`, tool, cid))
	busNumber := strings.Trim(adapterInfo, "\n")
	busNumber = fmt.Sprintf("%02s", busNumber)
	pwd := fmt.Sprintf(`/sys/bus/pci/devices/0000:%s:00.0/`, busNumber)
	fileList, err := os.ReadDir(pwd)
	if err != nil {
		// 如果无法读取目录，记录错误但继续处理
		fmt.Printf("Warning: failed to read directory %s: %v\n", pwd, err)
		return
	}
	for _, file := range fileList {
		switch {
		case strings.Contains(file.Name(), "host"):
			scsiBusNumber = strings.Replace(file.Name(), "host", "", -1)
		}
	}

	smartInfoSection := Bash(fmt.Sprintf(`smartctl /dev/bus/%s -d megaraid,%s -i`, scsiBusNumber, _deviceId))

	smartInfo := strings.Split(strings.Trim(smartInfoSection, "\n"), "\n")

	for _, v := range smartInfo {
		switch {
		case strings.Contains(v, "Vendor"):
			disk.Vendor = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "Device Model"):
			parts := strings.Split(strings.Trim(strings.Split(v, ":")[1], " "), " ")
			disk.Vendor = parts[0]
			if len(parts) > 1 {
				disk.Model = parts[1]
			} else {
				disk.Model = disk.Vendor
			}
		case strings.Contains(v, "Product"):
			disk.Model = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "User Capacity"):
			disk.Capacity = strings.Replace(strings.Split(strings.Trim(strings.Split(v, "[")[1], " "), "]")[0], ".00 ", " ", -1)
		case strings.Contains(strings.ToLower(v), strings.ToLower("Serial Number")):
			disk.SerialNumber = strings.Trim(strings.Split(v, ":")[1], " ")
		}
	}

	if strings.HasPrefix(strings.ToUpper(disk.Vendor), "ST") {
		disk.Vendor = "SEAGATE"
	}

	if strings.HasPrefix(strings.ToUpper(disk.Vendor), "HU") {
		disk.Vendor = "HGST"
	}

	if strings.HasPrefix(strings.ToUpper(disk.Vendor), "MICRON") {
		disk.Vendor = "MICRON"
	}

	if disk.State == "JBOD" {
		disk.Name = strings.Trim(Bash(fmt.Sprintf(`ls -l /dev/disk/by-id/ | grep -E "*%s*" | awk -F/ '{print $NF}'`, disk.SerialNumber)), "\n")
	}

	// 根据 PD 的 LD 信息精准匹配盘符与slot对应关系
	ldInfoSection := Bash(fmt.Sprintf(`%s -LdPdInfo -a%s | egrep "Virtual Drive|Sequence Number|%s"`, tool, cid, _wwn))

	ldInfo := strings.Split(strings.Trim(ldInfoSection, "\n"), "\n")

	for i, v := range ldInfo {
		var targetId, sequenceNum string

		// 如果当前行包含 _wwn，则在前后行查找 targetId 和 sequenceNum
		if strings.Contains(v, _wwn) {
			// 回溯查找最近的 Target Id，确保在 WWN 行之前存在
			for j := i - 1; j >= 0; j-- {
				if strings.Contains(ldInfo[j], "Target Id") {
					parts := strings.Split(strings.TrimSpace(ldInfo[j]), "(")
					if len(parts) > 1 {
						targetParts := strings.Split(strings.Trim(parts[1], " "), ":")
						if len(targetParts) > 1 {
							targetId = strings.Trim(targetParts[1], " )")
							break
						}
					}
				}
			}

			// 确保不越界
			if i+1 < len(ldInfo) {
				// 获取 sequenceNum，假设它在 _wwn 行的下一行
				sequenceParts := strings.Split(ldInfo[i+1], ":")
				if len(sequenceParts) > 1 {
					sequenceNum = strings.TrimSpace(sequenceParts[1])
				}
			}

			// 如果 targetId 和 sequenceNum 都找到了，执行查找
			if targetId != "" && sequenceNum != "" {
				disk.Name = strings.Trim(Bash(fmt.Sprintf(
					`ls -l /dev/disk/by-path/ | grep -E "pci-0000:%s:00.0-scsi-[0-9]:%s:%s:[0-9] " | awk -F/ '{print $NF}'`,
					busNumber, sequenceNum, targetId)), "\n")
			}
		}
	}

	if disk.Name == "" {
		disk.Name = "Nil"
	}

	// fmt.Printf("Device %s done\n", id)

	results <- disk
}

func (m *megacliCollector) Collect() []Disk {
	s := []Disk{}
	pdcesArray := []string{}
	c := controller.Collect()
	// fmt.Printf("server have %d controller\n", c.Num)
	for i := 0; i < c.Num; i++ {
		output := Bash(fmt.Sprintf(`%s -PDList -a%d | grep -E "Enclosure Device|Slot" | awk 'NR%%2==0{print a":"$NF}{a=$NF}' | awk '{print "%d:"$0}'`, c.Tool, i, i))
		pdces := strings.Split(strings.Trim(output, "\n"), "\n")
		// 过滤空字符串和格式不正确的条目
		for _, pd := range pdces {
			if pd != "" && strings.Count(pd, ":") == 2 {
				pdcesArray = append(pdcesArray, pd)
			}
		}
	}

	results := make(chan Disk, len(pdcesArray))

	var wg sync.WaitGroup

	for i := 0; i < len(pdcesArray); i++ {
		wg.Add(1)
		go megacli(pdcesArray[i], results, &wg)
	}

	wg.Wait()
	for i := 0; i < len(pdcesArray); i++ {
		s = append(s, <-results)
	}
	return s
}

func (m *megacliCollector) TurnOn(id string) error {
	c := controller.Collect()
	cid := strings.Split(id, ":")[0]
	eid := strings.Split(id, ":")[1]
	sid := strings.Split(id, ":")[2]
	locateInfo := Bash(fmt.Sprintf(`%s -PdLocate -start –physdrv[%s:%s] -a%s`, c.Tool, eid, sid, cid))
	return errors.New(locateInfo)
}

func (m *megacliCollector) TurnOff(id string) error {
	c := controller.Collect()
	cid := strings.Split(id, ":")[0]
	eid := strings.Split(id, ":")[1]
	sid := strings.Split(id, ":")[2]
	locateInfo := Bash(fmt.Sprintf(`%s -PdLocate -stop –physdrv[%s:%s] -a%s`, c.Tool, eid, sid, cid))
	return errors.New(locateInfo)
}
