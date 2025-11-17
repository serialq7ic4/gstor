package block

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/chenq7an/gstor/common/controller"
	"github.com/tidwall/gjson"
)

type storcliCollector struct{}

func formatBlockSize(block int) (size string) {
	if block < 1000 {
		return fmt.Sprintf("%.f B", float64(block)/float64(1))
	} else if block < (1000 * 1000) {
		return fmt.Sprintf("%.f KB", float64(block)/float64(1000))
	} else if block < (1000 * 1000 * 1000) {
		return fmt.Sprintf("%.f MB", float64(block)/float64(1000*1000))
	} else if block < (1000 * 1000 * 1000 * 1000) {
		return fmt.Sprintf("%.f GB", float64(block)/float64(1000*1000*1000))
	} else {
		return fmt.Sprintf("%.2f TB", float64(block)/float64(1000*1000*1000*1000))
	}
}

func storcli(id string, results chan<- Disk, wg *sync.WaitGroup) {
	tool := controller.StorcliPath
	defer wg.Done()

	// fmt.Printf("Device %s collecting\n", id)

	// 解析 ID，支持两种格式：c:e:s 和 c:s
	parts := strings.Split(id, ":")
	if len(parts) < 2 {
		fmt.Printf("Invalid device ID format: %s, expected format: c:e:s or c:s\n", id)
		return
	}

	var cid, eid, sid string
	if len(parts) == 2 {
		// 格式：c:s (没有 enclosure)
		cid = parts[0]
		eid = ""
		sid = parts[1]
	} else {
		// 格式：c:e:s
		cid = parts[0]
		eid = parts[1]
		sid = parts[2]
	}

	disk := Disk{Name: "", CES: id}
	// 从阵列卡 Pdinfo 中抓取的信息
	cmd := fmt.Sprintf(`%s /c%s/e%s/s%s show all`, tool, cid, eid, sid)
	vdcmd := fmt.Sprintf(`%s /c%s/vall show all J`, tool, cid)
	if eid == "" {
		cmd = fmt.Sprintf(`%s /c%s/s%s show all`, tool, cid, sid)
	}
	storcliInfo := Bash(cmd)
	storcliVDInfo := Bash(vdcmd)

	pdInfo := strings.Split(strings.Trim(storcliInfo, "\n"), "\n")
	// 解析 JSON 数据
	var vdInfo map[string]interface{}
	err := json.Unmarshal([]byte(storcliVDInfo), &vdInfo)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	for _, v := range pdInfo {
		switch {
		case strings.Contains(v, " SSD "):
			disk.State = strings.Trim(strings.Split(strings.Join(strings.Fields(v), " "), " ")[2], " ")
			disk.MediaType = strings.Trim(strings.Split(strings.Join(strings.Fields(v), " "), " ")[7], " ")
			disk.PDType = strings.Trim(strings.Split(strings.Join(strings.Fields(v), " "), " ")[6], " ")
		case strings.Contains(v, " HDD "):
			disk.State = strings.Trim(strings.Split(strings.Join(strings.Fields(v), " "), " ")[2], " ")
			disk.MediaType = strings.Trim(strings.Split(strings.Join(strings.Fields(v), " "), " ")[7], " ")
			disk.PDType = strings.Trim(strings.Split(strings.Join(strings.Fields(v), " "), " ")[6], " ")
		case strings.Contains(v, "Media Error Count"):
			disk.MediaError = strings.Trim(strings.Split(v, "=")[1], " ")
		case strings.Contains(v, "Predictive Failure Count"):
			disk.PredictError = strings.Trim(strings.Split(v, "=")[1], " ")
		case strings.Contains(v, "SN ="):
			disk.SerialNumber = strings.Trim(strings.Split(v, "=")[1], " ")
		case strings.Contains(v, "Model Number"):
			parts := strings.Split(strings.Trim(strings.Split(v, "=")[1], " "), " ")
			disk.Vendor = parts[0]
			if len(parts) > 1 {
				disk.Model = parts[1]
			} else {
				disk.Model = disk.Vendor
			}
		case strings.Contains(v, "Raw size"):
			sectors := strings.Split(strings.Trim(strings.Split(strings.Trim(strings.Split(v, "[")[1], " "), " ")[0], " "), " ")[0]
			blocks, _ := strconv.ParseInt(sectors, 0, 64)
			disk.Capacity = strings.Replace(formatBlockSize(int(blocks)*512), ".00", "", -1)
		}
	}

	targetEIDSlt := fmt.Sprintf("%s:%s", eid, sid)
	if disk.State == "Onln" {
		// 使用 GJSON 查询符合条件的 PD，并获取对应的 VD ID
		controllers := gjson.Get(storcliVDInfo, "Controllers.#.Response Data")
		var vdID string
		var scsiNaaIdStr string
		controllers.ForEach(func(key, value gjson.Result) bool {
			value.ForEach(func(k, v gjson.Result) bool {
				if v.IsArray() {
					v.ForEach(func(_, pd gjson.Result) bool {
						if pd.Get("EID:Slt").String() == targetEIDSlt {
							// 查找对应的 VD ID
							vdID = strings.ReplaceAll(strings.TrimPrefix(k.String(), "PDs for "), " ", "")
							return false
						}
						return true
					})
				}
				return true
			})
			return true
		})
		// 如果找到 VD ID，则查询对应的 SCSI NAA Id
		if vdID != "" {
			scsiNaaPath := fmt.Sprintf(`Controllers.#.Response Data.%s Properties.SCSI NAA Id`, vdID)
			scsiNaaId := gjson.Get(storcliVDInfo, scsiNaaPath).Array()
			// 将 SCSI NAA Id 转换为字符串
			if len(scsiNaaId) > 0 {
				scsiNaaIdStr = scsiNaaId[0].String()
				disk.Name = strings.Trim(Bash(fmt.Sprintf(
					`ls -l /dev/disk/by-id/ | grep "%s" | grep -v part | awk -F/ '{print $NF}' | sort | uniq`,
					scsiNaaIdStr)), "\n")
			} else {
				fmt.Println("No SCSI NAA Id found")
			}
		} else {
			// 兼容 3008 不支持看vd
			disk.Name = "sda"
		}
	} else {
		lsblkInfoSection := Bash(`lsblk -o KNAME,MODEL,SERIAL,TYPE | grep disk | grep ^sd[a-z] | grep -vi "logical"`)

		lsblkInfo := strings.Split(strings.Trim(lsblkInfoSection, "\n"), "\n")

		for _, v := range lsblkInfo {
			switch {
			case strings.Contains(v, disk.SerialNumber):
				disk.Name = strings.Trim(strings.Split(strings.Join(strings.Fields(v), ":"), ":")[0], " ")
			}
		}

	}

	if disk.Vendor == "" {
		if disk.Name != "" {
			model := Bash(fmt.Sprintf(`smartctl -i /dev/%s | egrep "Device Model|Vendor"`, disk.Name))
			disk.Vendor = strings.Trim(strings.Split(strings.Trim(strings.Split(model, ":")[1], " "), " ")[0], " ")
		} else {
			disk.Vendor = "unknown"
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

	// fmt.Printf("Device %s done\n", id)

	results <- disk
}

func (m *storcliCollector) Collect() []Disk {
	s := []Disk{}
	pdcesArray := []string{}
	c := controller.Collect()
	// fmt.Printf("server have %d controller\n", c.Num)
	for i := 0; i < c.Num; i++ {
		output := Bash(fmt.Sprintf(`%s /c%d show | egrep "SSD|HDD" | awk '{print "%d:"$1}' | sort | uniq`, c.Tool, i, i))
		pdces := strings.Split(strings.Trim(output, "\n"), "\n")
		// 过滤空字符串
		for _, pd := range pdces {
			if pd != "" && strings.Contains(pd, ":") {
				pdcesArray = append(pdcesArray, pd)
			}
		}
	}

	results := make(chan Disk, len(pdcesArray))

	var wg sync.WaitGroup

	for i := 0; i < len(pdcesArray); i++ {
		wg.Add(1)
		go storcli(pdcesArray[i], results, &wg)
	}

	wg.Wait()
	for i := 0; i < len(pdcesArray); i++ {
		s = append(s, <-results)
	}
	return s
}

func (m *storcliCollector) TurnOn(id string) error {
	c := controller.Collect()
	cid := strings.Split(id, ":")[0]
	eid := strings.Split(id, ":")[1]
	sid := strings.Split(id, ":")[2]
	cmd := fmt.Sprintf(`%s /c%s/e%s/s%s start locate`, c.Tool, cid, eid, sid)
	if eid == "" {
		cmd = fmt.Sprintf(`%s /c%s/s%s start locate`, c.Tool, cid, sid)
	}
	locateInfo := Bash(cmd)
	return errors.New(locateInfo)
}

func (m *storcliCollector) TurnOff(id string) error {
	c := controller.Collect()
	cid := strings.Split(id, ":")[0]
	eid := strings.Split(id, ":")[1]
	sid := strings.Split(id, ":")[2]
	cmd := fmt.Sprintf(`%s /c%s/e%s/s%s stop locate`, c.Tool, cid, eid, sid)
	if eid == "" {
		cmd = fmt.Sprintf(`%s /c%s/s%s stop locate`, c.Tool, cid, sid)
	}
	locateInfo := Bash(cmd)
	return errors.New(locateInfo)
}
