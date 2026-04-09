package block

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/chenq7an/gstor/common/controller"
	"github.com/chenq7an/gstor/common/utils"
	"github.com/tidwall/gjson"
)

type storcliCollector struct{}

type storcliBlockDevice struct {
	Name      string
	SizeBytes uint64
	Type      string
	Model     string
	Vendor    string
	Serial    string
}

type storcliControllerVDInfo struct {
	Output    string
	Supported bool
}

type storcliControllerSnapshot struct {
	DeviceIDs []string
	Disks     map[string]Disk
}

var storcliLSBLKPairPattern = regexp.MustCompile(`([A-Z]+)="([^"]*)"`)
var storcliDriveSectionPattern = regexp.MustCompile(`^Drive /c(\d+)(?:/e(\d+))?/s(\d+) :$`)

func execStorcliCommand(cmd string) string {
	output, err := utils.ExecShell(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: storcli command failed: %v\n", err)
		return ""
	}
	return output
}

func execStorcliDiscoveryCommand(cmd string) string {
	result, err := utils.ExecShellResult(cmd)
	if err != nil {
		if strings.Contains(result.Output, "No Enclosure found") {
			return result.Output
		}
		fmt.Fprintf(os.Stderr, "Warning: storcli discovery command failed: %v\n", err)
		return result.Output
	}
	return result.Output
}

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

func storcli(id string, controllerDisks map[string]map[string]Disk, blockDevices []storcliBlockDevice, controllerVDInfos map[string]storcliControllerVDInfo, results chan<- Disk, wg *sync.WaitGroup) {
	defer wg.Done()

	utils.DebugLogStep("开始收集设备信息: %s", id)

	// 解析 ID，支持两种格式：c:e:s 和 c:s
	parts := strings.Split(id, ":")
	if len(parts) < 2 {
		fmt.Fprintf(os.Stderr, "Invalid device ID format: %s, expected format: c:e:s or c:s\n", id)
		return
	}

	var cid, eid, sid string
	if len(parts) == 2 {
		// 格式：c:s (没有 enclosure)
		cid = parts[0]
		eid = ""
		sid = parts[1]
		utils.DebugLog("解析设备 ID: controller=%s, slot=%s (无 enclosure)", cid, sid)
	} else {
		// 格式：c:e:s
		cid = parts[0]
		eid = parts[1]
		sid = parts[2]
		utils.DebugLog("解析设备 ID: controller=%s, enclosure=%s, slot=%s", cid, eid, sid)
	}

	disk := Disk{Name: "", CES: id, MediaError: "0", PredictError: "0"}
	if disksByController, ok := controllerDisks[cid]; ok {
		if parsedDisk, ok := disksByController[id]; ok {
			disk = parsedDisk
		}
	}

	// 获取盘符：优先使用序列号匹配
	if disk.SerialNumber != "" {
		utils.DebugLogStep("通过序列号匹配盘符: %s", disk.SerialNumber)
		disk.Name = resolveStorcliDeviceNameBySerial(blockDevices, disk.SerialNumber)
	}

	// 如果序列号匹配失败，且是 Onln 状态，尝试通过 VD 信息获取
	if disk.Name == "" && disk.State == "Onln" && eid != "" {
		disk.Name = resolveStorcliDeviceNameByVDInfo(controllerVDInfos[cid], fmt.Sprintf("%s:%s", eid, sid))
	}

	if disk.Vendor == "" {
		if disk.Name != "" {
			model := execStorcliCommand(fmt.Sprintf(`smartctl -i /dev/%s | egrep "Device Model|Vendor"`, disk.Name))
			disk.Vendor = strings.Trim(strings.Split(strings.Trim(strings.Split(model, ":")[1], " "), " ")[0], " ")
		} else {
			disk.Vendor = "unknown"
		}
	}

	disk.Vendor = NormalizeVendor(disk.Vendor)

	// fmt.Printf("Device %s done\n", id)

	results <- disk
}

func (m *storcliCollector) Collect() []Disk {
	s := []Disk{}
	pdcesArray := []string{}
	c := controller.Collect()
	blockDevices := collectStorcliBlockDevices()
	controllerDisks := make(map[string]map[string]Disk, c.Num)
	controllerVDInfos := make(map[string]storcliControllerVDInfo, c.Num)
	utils.DebugLogStep("开始收集 storcli 设备列表，控制器数量: %d", c.Num)

	for i := 0; i < c.Num; i++ {
		utils.DebugLogStep("获取控制器 %d 的所有物理磁盘", i)
		snapshot := loadStorcliControllerSnapshot(c.Tool, i)
		discovered := snapshot.DeviceIDs
		controllerDisks[strconv.Itoa(i)] = snapshot.Disks
		utils.DebugLog("控制器 %d 共发现 %d 个设备", i, len(discovered))
		for _, deviceID := range discovered {
			pdcesArray = append(pdcesArray, deviceID)
			utils.DebugLog("发现设备: %s", deviceID)
		}
		if hasStorcliEnclosureDeviceIDs(discovered) {
			controllerVDInfos[strconv.Itoa(i)] = loadStorcliControllerVDInfo(c.Tool, i)
		}
	}

	utils.DebugLogStep("共发现 %d 个设备，开始并发收集详细信息", len(pdcesArray))
	results := make(chan Disk, len(pdcesArray))

	var wg sync.WaitGroup

	for i := 0; i < len(pdcesArray); i++ {
		wg.Add(1)
		go storcli(pdcesArray[i], controllerDisks, blockDevices, controllerVDInfos, results, &wg)
	}

	wg.Wait()
	for i := 0; i < len(pdcesArray); i++ {
		s = append(s, <-results)
	}
	fillStorcliLogicalVolumeNames(s, blockDevices)
	return s
}

func (m *storcliCollector) TurnOn(id string) error {
	slot, err := ParseSlotID(id)
	if err != nil {
		return err
	}

	c := controller.Collect()
	cmd := fmt.Sprintf(`%s /c%s/e%s/s%s start locate`, c.Tool, slot.ControllerID, slot.EnclosureID, slot.SlotID)
	if !slot.HasEnclosure() {
		cmd = fmt.Sprintf(`%s /c%s/s%s start locate`, c.Tool, slot.ControllerID, slot.SlotID)
	}
	_, err = utils.ExecShell(cmd)
	return err
}

func discoverStorcliDeviceIDs(tool string, controllerID int) []string {
	eallOutput := execStorcliDiscoveryCommand(fmt.Sprintf(`%s /c%d/eall/sall show`, tool, controllerID))
	sallOutput := execStorcliDiscoveryCommand(fmt.Sprintf(`%s /c%d/sall show`, tool, controllerID))
	return discoverStorcliDeviceIDsFromOutputs(eallOutput, sallOutput, controllerID)
}

func loadStorcliControllerSnapshot(tool string, controllerID int) storcliControllerSnapshot {
	eallOutput := execStorcliDiscoveryCommand(fmt.Sprintf(`%s /c%d/eall/sall show all`, tool, controllerID))
	snapshot := parseStorcliControllerSnapshot(eallOutput, controllerID)
	if len(snapshot.DeviceIDs) > 0 {
		return snapshot
	}

	sallOutput := execStorcliDiscoveryCommand(fmt.Sprintf(`%s /c%d/sall show all`, tool, controllerID))
	return parseStorcliControllerSnapshot(sallOutput, controllerID)
}

func parseStorcliControllerSnapshot(output string, controllerID int) storcliControllerSnapshot {
	snapshot := storcliControllerSnapshot{
		DeviceIDs: make([]string, 0),
		Disks:     make(map[string]Disk),
	}
	currentID := ""

	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		if sectionID, ok := parseStorcliDriveSectionID(line, controllerID); ok {
			currentID = sectionID
			if _, exists := snapshot.Disks[currentID]; !exists {
				snapshot.DeviceIDs = append(snapshot.DeviceIDs, currentID)
				snapshot.Disks[currentID] = Disk{
					Name:         "",
					CES:          currentID,
					MediaError:   "0",
					PredictError: "0",
				}
			}
			continue
		}
		if currentID == "" {
			continue
		}

		disk := snapshot.Disks[currentID]
		applyStorcliControllerLine(&disk, line)
		snapshot.Disks[currentID] = disk
	}

	return snapshot
}

func discoverStorcliDeviceIDsFromOutputs(eallOutput string, sallOutput string, controllerID int) []string {
	if deviceIDs := parseStorcliDiscoveryOutput(eallOutput, controllerID); len(deviceIDs) > 0 {
		return deviceIDs
	}
	return parseStorcliDiscoveryOutput(sallOutput, controllerID)
}

func parseStorcliDiscoveryOutput(output string, controllerID int) []string {
	deviceIDs := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		slotToken, ok := normalizeStorcliSlotToken(fields[0])
		if !ok {
			continue
		}
		deviceIDs = append(deviceIDs, fmt.Sprintf("%d:%s", controllerID, slotToken))
	}
	return deviceIDs
}

func parseStorcliDriveSectionID(line string, controllerID int) (string, bool) {
	matches := storcliDriveSectionPattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) != 4 {
		return "", false
	}

	if matches[2] != "" {
		return fmt.Sprintf("%d:%s:%s", controllerID, matches[2], matches[3]), true
	}
	return fmt.Sprintf("%d:%s", controllerID, matches[3]), true
}

func applyStorcliControllerLine(disk *Disk, line string) {
	fields := strings.Fields(line)
	if len(fields) >= 8 {
		if _, ok := normalizeStorcliSlotToken(fields[0]); ok {
			disk.State = fields[2]
			disk.PDType = fields[6]
			disk.MediaType = fields[7]

			lastField := fields[len(fields)-1]
			if lastField == "JBOD" || lastField == "UGood" || lastField == "UBad" {
				disk.State = lastField
			}
			return
		}
	}

	switch {
	case strings.Contains(line, "Media Error Count"):
		disk.MediaError = strings.TrimSpace(strings.Split(line, "=")[1])
	case strings.Contains(line, "Predictive Failure Count"):
		disk.PredictError = strings.TrimSpace(strings.Split(line, "=")[1])
	case strings.Contains(line, "SN ="):
		disk.SerialNumber = strings.TrimSpace(strings.Split(line, "=")[1])
	case strings.Contains(line, "Model Number"):
		parts := strings.Fields(strings.TrimSpace(strings.Split(line, "=")[1]))
		if len(parts) > 0 {
			disk.Vendor = parts[0]
			if len(parts) > 1 {
				disk.Model = parts[1]
			} else {
				disk.Model = disk.Vendor
			}
		}
	case strings.Contains(line, "Raw size"):
		sectors := strings.Split(strings.Trim(strings.Split(strings.Trim(strings.Split(line, "[")[1], " "), " ")[0], " "), " ")[0]
		blocks, err := strconv.ParseInt(sectors, 0, 64)
		if err == nil {
			disk.Capacity = strings.Replace(formatBlockSize(int(blocks)*512), ".00", "", -1)
		}
	}
}

func normalizeStorcliSlotToken(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}

	if strings.HasPrefix(trimmed, ":") {
		slot := strings.TrimPrefix(trimmed, ":")
		if isNumericString(slot) {
			return slot, true
		}
		return "", false
	}

	if strings.Count(trimmed, ":") != 1 {
		return "", false
	}

	parts := strings.SplitN(trimmed, ":", 2)
	if isNumericString(parts[0]) && isNumericString(parts[1]) {
		return trimmed, true
	}
	return "", false
}

func isNumericString(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func collectStorcliBlockDevices() []storcliBlockDevice {
	output, err := utils.ExecShell(`lsblk -bdn -P -o KNAME,SIZE,TYPE,MODEL,VENDOR,SERIAL`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to collect block devices for storcli mapping: %v\n", err)
		return nil
	}

	devices := make([]storcliBlockDevice, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := make(map[string]string, 6)
		for _, match := range storcliLSBLKPairPattern.FindAllStringSubmatch(line, -1) {
			fields[match[1]] = match[2]
		}
		sizeBytes, err := strconv.ParseUint(fields["SIZE"], 10, 64)
		if err != nil {
			continue
		}

		devices = append(devices, storcliBlockDevice{
			Name:      fields["KNAME"],
			SizeBytes: sizeBytes,
			Type:      fields["TYPE"],
			Model:     fields["MODEL"],
			Vendor:    fields["VENDOR"],
			Serial:    fields["SERIAL"],
		})
	}
	return devices
}

func resolveStorcliDeviceNameBySerial(blockDevices []storcliBlockDevice, serial string) string {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return ""
	}

	for _, device := range blockDevices {
		if device.Type != "disk" || !strings.HasPrefix(device.Name, "sd") || device.isSharedLogicalVolume() {
			continue
		}
		deviceSerial := strings.TrimSpace(device.Serial)
		if deviceSerial == "" {
			continue
		}
		if deviceSerial == serial || strings.Contains(deviceSerial, serial) {
			return device.Name
		}
	}
	return ""
}

func hasStorcliEnclosureDeviceIDs(deviceIDs []string) bool {
	for _, deviceID := range deviceIDs {
		if strings.Count(deviceID, ":") == 2 {
			return true
		}
	}
	return false
}

func loadStorcliControllerVDInfo(tool string, controllerID int) storcliControllerVDInfo {
	output := execStorcliCommand(fmt.Sprintf(`%s /c%d/vall show all J`, tool, controllerID))
	return parseStorcliControllerVDInfo(output)
}

func parseStorcliControllerVDInfo(output string) storcliControllerVDInfo {
	output = strings.TrimSpace(output)
	if output == "" || strings.Contains(output, "Un-supported command") {
		return storcliControllerVDInfo{}
	}
	return storcliControllerVDInfo{
		Output:    output,
		Supported: true,
	}
}

func resolveStorcliDeviceNameByVDInfo(vdInfo storcliControllerVDInfo, targetEIDSlt string) string {
	if !vdInfo.Supported || targetEIDSlt == "" {
		return ""
	}

	controllers := gjson.Get(vdInfo.Output, "Controllers.#.Response Data")
	var vdID string
	controllers.ForEach(func(_, value gjson.Result) bool {
		value.ForEach(func(k, v gjson.Result) bool {
			if !v.IsArray() {
				return true
			}
			v.ForEach(func(_, pd gjson.Result) bool {
				if pd.Get("EID:Slt").String() == targetEIDSlt {
					vdID = strings.ReplaceAll(strings.TrimPrefix(k.String(), "PDs for "), " ", "")
					return false
				}
				return true
			})
			return vdID == ""
		})
		return vdID == ""
	})
	if vdID == "" {
		return ""
	}

	scsiNaaPath := fmt.Sprintf(`Controllers.#.Response Data.%s Properties.SCSI NAA Id`, vdID)
	scsiNaaIDs := gjson.Get(vdInfo.Output, scsiNaaPath).Array()
	if len(scsiNaaIDs) == 0 {
		return ""
	}

	scsiNaaID := scsiNaaIDs[0].String()
	if scsiNaaID == "" {
		return ""
	}

	return strings.Trim(execStorcliCommand(fmt.Sprintf(
		`ls -l /dev/disk/by-id/ | grep "%s" | grep -v part | awk -F/ '{print $NF}' | sort | uniq`,
		scsiNaaID)), "\n")
}

func fillStorcliLogicalVolumeNames(disks []Disk, blockDevices []storcliBlockDevice) {
	if len(disks) == 0 || len(blockDevices) == 0 {
		return
	}

	usedNames := make(map[string]struct{}, len(disks))
	unnamedOnln := make([]int, 0)
	for i, disk := range disks {
		if disk.Name != "" {
			usedNames[disk.Name] = struct{}{}
			continue
		}
		if disk.State == "Onln" {
			unnamedOnln = append(unnamedOnln, i)
		}
	}
	if len(unnamedOnln) < 2 {
		return
	}

	candidates := make([]storcliBlockDevice, 0, 1)
	for _, device := range blockDevices {
		if device.Type != "disk" || !strings.HasPrefix(device.Name, "sd") {
			continue
		}
		if _, ok := usedNames[device.Name]; ok {
			continue
		}
		if !device.isSharedLogicalVolume() {
			continue
		}
		candidates = append(candidates, device)
	}
	if len(candidates) != 1 {
		return
	}

	candidate := candidates[0]
	for _, idx := range unnamedOnln {
		sizeBytes, ok := parseStorcliCapacityBytes(disks[idx].Capacity)
		if !ok || !storcliCapacitiesRoughlyMatch(sizeBytes, candidate.SizeBytes) {
			return
		}
	}

	for _, idx := range unnamedOnln {
		disks[idx].Name = candidate.Name
	}
}

func (device storcliBlockDevice) isSharedLogicalVolume() bool {
	model := strings.ToLower(strings.TrimSpace(device.Model))
	vendor := strings.ToLower(strings.TrimSpace(device.Vendor))
	return strings.Contains(model, "logical volume") || vendor == "lsi"
}

func parseStorcliCapacityBytes(value string) (uint64, bool) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) != 2 {
		return 0, false
	}

	number, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, false
	}

	multiplier := uint64(0)
	switch strings.ToUpper(fields[1]) {
	case "B":
		multiplier = 1
	case "KB":
		multiplier = 1000
	case "MB":
		multiplier = 1000 * 1000
	case "GB":
		multiplier = 1000 * 1000 * 1000
	case "TB":
		multiplier = 1000 * 1000 * 1000 * 1000
	default:
		return 0, false
	}

	return uint64(number * float64(multiplier)), true
}

func storcliCapacitiesRoughlyMatch(left uint64, right uint64) bool {
	if left == 0 || right == 0 {
		return false
	}

	var diff uint64
	if left > right {
		diff = left - right
	} else {
		diff = right - left
	}

	maxValue := left
	if right > maxValue {
		maxValue = right
	}

	return diff*100 <= maxValue*10
}

func (m *storcliCollector) TurnOff(id string) error {
	slot, err := ParseSlotID(id)
	if err != nil {
		return err
	}

	c := controller.Collect()
	cmd := fmt.Sprintf(`%s /c%s/e%s/s%s stop locate`, c.Tool, slot.ControllerID, slot.EnclosureID, slot.SlotID)
	if !slot.HasEnclosure() {
		cmd = fmt.Sprintf(`%s /c%s/s%s stop locate`, c.Tool, slot.ControllerID, slot.SlotID)
	}
	_, err = utils.ExecShell(cmd)
	return err
}
