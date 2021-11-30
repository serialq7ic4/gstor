package block

import (
	"fmt"
	"github.com/chenq7an/gstor/common/controller"
	"strconv"
	"strings"
	"sync"
)

type arcconfCollector struct{}

func formatDiskSize(kb int) (size string) {
	if kb < 1000 {
		return fmt.Sprintf("%.f KB", float64(kb)/float64(1))
	} else if kb < (1000 * 1000) {
		return fmt.Sprintf("%.f MB", float64(kb)/float64(1000))
	} else if kb < (1000 * 1000 * 1000) {
		return fmt.Sprintf("%.f GB", float64(kb)/float64(1000*1000))
	} else {
		return fmt.Sprintf("%.2f TB", float64(kb)/float64(1000*1000*1000))
	}
}

func arcconf(id string, results chan<- *Disk, wg *sync.WaitGroup) {

	tool := "/usr/sbin/arcconf"
	defer wg.Done()

	fmt.Printf("Device %s collecting\n", id)

	cid := strings.Split(id, ":")[0]
	eid := strings.Split(id, ":")[1]
	sid := strings.Split(id, ":")[2]

	disk := &Disk{CES: id}
	// 从阵列卡 Pdinfo 中抓取的信息
	arcconfInfo := Bash(fmt.Sprintf(`%s getconfig %s pd %s %s | egrep "  State|Model|Serial number|Total Size|SSD|Medium Error Count|SMART Warning Count"`, tool, cid, eid, sid))

	pdInfo := strings.Split(strings.Trim(arcconfInfo, "\n"), "\n")

	for _, v := range pdInfo {
		switch {
		case strings.Contains(v, "State"):
			disk.State = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "Model"):
			disk.Vendor = strings.Trim(strings.Split(strings.Trim(strings.Split(v, ":")[1], " "), " ")[0], " ")
		case strings.Contains(v, "Serial number"):
			disk.SerialNumber = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "Total Size"):
			size := strings.Trim(strings.Split(strings.Trim(strings.Split(v, ":")[1], " "), " ")[0], " ")
			sizeofMB, _ := strconv.Atoi(size)
			disk.Capacity = strings.Replace(formatDiskSize(sizeofMB/1000*1024*1024), ".00", "", -1)
		case strings.Contains(v, "No"):
			disk.MediaType = "HDD"
		case strings.Contains(v, "Yes"):
			disk.MediaType = "SSD"
		case strings.Contains(v, "Medium Error Count"):
			disk.MediaError = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "SMART Warning Count"):
			disk.PredictError = strings.Trim(strings.Split(v, ":")[1], " ")
		default:
			fmt.Println(v)
		}
	}

	//从 PD 的 LD 中抓取的信息
	lsscsiInfoSection := Bash(`lsscsi | grep dev | awk '{print $4,$NF}'`)

	lsscsiInfo := strings.Split(strings.Trim(lsscsiInfoSection, "\n"), "\n")

	logicalDeviceName := strings.Trim(Bash(fmt.Sprintf(`%s getconfig %s ld | egrep "Logical Device name|%s" | grep -B1 %s | grep "Logical Device name" | awk '{print $NF}'`, tool, cid, disk.SerialNumber, disk.SerialNumber)), "\n")

	for i, v := range lsscsiInfo {
		switch {
		case strings.HasPrefix(v, fmt.Sprintf("%s ", logicalDeviceName)):
			disk.Name = strings.Trim(strings.Split(lsscsiInfo[i], "/")[2], " ")
		}
	}

	if disk.Name == "" {
		disk.Name = "Nil"
	}

	if strings.HasPrefix(disk.Vendor, "ST") {
		disk.Vendor = "SEAGATE"
	}

	if strings.HasPrefix(disk.Vendor, "HUS") {
		disk.Vendor = "HGST"
	}

	fmt.Printf("Device %s done\n", id)

	results <- disk
}

func (m *arcconfCollector) Collect() []*Disk {
	s := make([]*Disk, 0)
	pdcesArray := make([]string, 0)
	c := controller.Collect()
	// fmt.Printf("server have %d controller\n", c.Num)
	for i := 1; i <= c.Num; i++ {
		output := Bash(fmt.Sprintf(`%s list %d | grep Physical | grep Drive | awk '{print $2}' | awk -F, '{print "%d:"$1":"$2}'`, c.Tool, i, i))
		pdces := strings.Split(strings.Trim(output, "\n"), "\n")
		// fmt.Println(pdces)
		pdcesArray = append(pdcesArray, pdces...)
	}

	results := make(chan *Disk, len(pdcesArray))

	var wg sync.WaitGroup

	for i := 0; i < len(pdcesArray); i++ {
		wg.Add(1)
		go arcconf(pdcesArray[i], results, &wg)
	}

	wg.Wait()
	for i := 0; i < len(pdcesArray); i++ {
		s = append(s, <-results)
	}
	return s
}
