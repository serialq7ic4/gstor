package block

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"

	"github.com/chenq7an/gstor/common/controller"
	"github.com/spf13/cast"
)

type megacliCollector struct{}

func megacli(id string, results chan<- Disk, wg *sync.WaitGroup) {

	var _deviceId string
	var _wwn string

	tool := "/opt/MegaRAID/MegaCli/MegaCli64"
	defer wg.Done()

	// fmt.Printf("Device %s collecting\n", id)

	cid := strings.Split(id, ":")[0]
	eid := strings.Split(id, ":")[1]
	sid := strings.Split(id, ":")[2]

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

	//从 SMART 中抓取的信息
	var scsiBusNumber string
	adapterInfo := Bash(fmt.Sprintf(`%s -adpgetpciinfo -a%s | grep "Bus Number" | awk '{print $NF}'`, tool, cid))
	busNumber := strings.Trim(adapterInfo, "\n")
	busNumber = fmt.Sprintf("%02s", busNumber)
	pwd := fmt.Sprintf(`/sys/bus/pci/devices/0000:%s:00.0/`, busNumber)
	fileList, err := ioutil.ReadDir(pwd)
	if err != nil {
		log.Fatal(err)
	}
	for i := range fileList {
		switch {
		case strings.Contains(fileList[i].Name(), "host"):
			scsiBusNumber = strings.Replace(fileList[i].Name(), "host", "", -1)
		}
	}

	smartInfoSection := Bash(fmt.Sprintf(`smartctl /dev/bus/%s -d megaraid,%s -i`, scsiBusNumber, _deviceId))

	smartInfo := strings.Split(strings.Trim(smartInfoSection, "\n"), "\n")

	for _, v := range smartInfo {
		switch {
		case strings.Contains(v, "Vendor"):
			disk.Vendor = strings.Trim(strings.Split(v, ":")[1], " ")
		case strings.Contains(v, "Device Model"):
			disk.Vendor = strings.Split(strings.Trim(strings.Split(v, ":")[1], " "), " ")[0]
		case strings.Contains(v, "User Capacity"):
			disk.Capacity = strings.Replace(strings.Split(strings.Trim(strings.Split(v, "[")[1], " "), "]")[0], ".00 ", " ", -1)
		case strings.Contains(strings.ToLower(v), strings.ToLower("Serial Number")):
			disk.SerialNumber = strings.Trim(strings.Split(v, ":")[1], " ")
		}
	}

	if strings.HasPrefix(disk.Vendor, "ST") {
		disk.Vendor = "SEAGATE"
	}

	if strings.HasPrefix(disk.Vendor, "HUS") {
		disk.Vendor = "HGST"
	}

	if strings.HasPrefix(disk.Vendor, "MICRON") {
		disk.Vendor = "Micron"
	}

	//根据 PD 的 LD 信息精准匹配盘符与slot对应关系
	ldInfoSection := Bash(fmt.Sprintf(`%s -LdPdInfo -a%s | egrep "Virtual Drive|%s"`, tool, cid, _wwn))

	ldInfo := strings.Split(strings.Trim(ldInfoSection, "\n"), "\n")

	for i, v := range ldInfo {
		switch {
		case strings.Contains(v, _wwn):
			i = i - 1
			targetId := strings.Split(strings.Trim(strings.Split(strings.Trim(strings.Split(ldInfo[i], "(")[1], " "), ":")[1], " "), ")")[0]
			disk.Name = strings.Trim(Bash(fmt.Sprintf(`ls -l /dev/disk/by-path/ | grep -E "pci-0000:%s:00.0-scsi-[0-9]:[0-9]:%s:[0-9] " | awk -F/ '{print $NF}'`, busNumber, targetId)), "\n")
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
		// fmt.Println(pdces)
		pdcesArray = append(pdcesArray, pdces...)
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
	s = append(s, Nvme()...)
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
