package block

import (
	"fmt"
	"github.com/chenq7an/gstor/common/controller"
	"strconv"
	"strings"
	"sync"
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

func storcli(id string, results chan<- *Disk, wg *sync.WaitGroup) {

	tool := "/opt/MegaRAID/storcli/storcli64"
	defer wg.Done()

	// fmt.Printf("Device %s collecting\n", id)

	cid := strings.Split(id, ":")[0]
	eid := strings.Split(id, ":")[1]
	sid := strings.Split(id, ":")[2]

	disk := &Disk{Name: "sdb", CES: id}
	// 从阵列卡 Pdinfo 中抓取的信息
	cmd := fmt.Sprintf(`%s /c%s/e%s/s%s show all`, tool, cid, eid, sid)
	if eid == "" {
		cmd = fmt.Sprintf(`%s /c%s/s%s show all`, tool, cid, sid)
	}
	storcliInfo := Bash(cmd)

	pdInfo := strings.Split(strings.Trim(storcliInfo, "\n"), "\n")

	for _, v := range pdInfo {
		switch {
		case strings.Contains(v, "SATA"):
			disk.State = strings.Trim(strings.Split(strings.Join(strings.Fields(v), " "), " ")[2], " ")
			disk.MediaType = strings.Trim(strings.Split(strings.Join(strings.Fields(v), " "), " ")[7], " ")
		case strings.Contains(v, "Media Error Count"):
			disk.MediaError = strings.Trim(strings.Split(v, "=")[1], " ")
		case strings.Contains(v, "Predictive Failure Count"):
			disk.PredictError = strings.Trim(strings.Split(v, "=")[1], " ")
		case strings.Contains(v, "SN ="):
			disk.SerialNumber = strings.Trim(strings.Split(v, "=")[1], " ")
		case strings.Contains(v, "Model Number"):
			disk.Vendor = strings.Split(strings.Trim(strings.Split(v, "=")[1], " "), " ")[0]
		// case strings.Contains(v, "Number of Blocks"):
		// blocks, _ := strconv.Atoi(strings.Trim(strings.Split(v, "=")[1], " "))
		// disk.Capacity = strings.Replace(formatBlockSize(blocks*512), ".00", "", -1)
		case strings.Contains(v, "Raw size"):
			sectors := strings.Split(strings.Trim(strings.Split(strings.Trim(strings.Split(v, "[")[1], " "), " ")[0], " "), " ")[0]
			blocks, _ := strconv.ParseInt(sectors, 0, 64)
			disk.Capacity = strings.Replace(formatBlockSize(int(blocks)*512), ".00", "", -1)
		}
	}

	if disk.State == "Onln" {
		disk.Name = "sda"
	} else {
		lsblkInfoSection := Bash(`lsblk -o KNAME,MODEL,SERIAL,TYPE | grep disk | grep ^sd[a-z]`)

		lsblkInfo := strings.Split(strings.Trim(lsblkInfoSection, "\n"), "\n")

		for _, v := range lsblkInfo {
			switch {
			case strings.Contains(v, disk.SerialNumber):
				disk.Name = strings.Trim(strings.Split(strings.Join(strings.Fields(v), ":"), ":")[0], " ")
			}
		}

	}

	if disk.Vendor == "" {
		model := Bash(fmt.Sprintf(`smartctl -i /dev/%s | egrep "Device Model|Vendor"`, disk.Name))
		disk.Vendor = strings.Trim(strings.Split(strings.Trim(strings.Split(model, ":")[1], " "), " ")[0], " ")
	}

	if strings.HasPrefix(disk.Vendor, "ST") {
		disk.Vendor = "SEAGATE"
	}
	// fmt.Printf("Device %s done\n", id)

	results <- disk
}

func (m *storcliCollector) Collect() []*Disk {
	s := make([]*Disk, 0)
	pdcesArray := make([]string, 0)
	c := controller.Collect()
	// fmt.Printf("server have %d controller\n", c.Num)
	for i := 0; i < c.Num; i++ {
		output := Bash(fmt.Sprintf(`%s /c%d show | egrep "SATA SSD|SATA HDD" | awk '{print "%d:"$1}'`, c.Tool, i, i))
		pdces := strings.Split(strings.Trim(output, "\n"), "\n")
		// fmt.Println(pdces)
		pdcesArray = append(pdcesArray, pdces...)
	}

	results := make(chan *Disk, len(pdcesArray))

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
