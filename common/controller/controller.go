package controller

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chenq7an/gstor/common/utils"
)

type Controller struct {
	Name  string
	Num   int
	Tool  string
	Avail bool
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// bash 执行 shell 命令，使用 /bin/bash
// 移除所有换行符以保持与原有行为一致
func bash(cmd string) (string, error) {
	result, err := utils.ExecShellWithShell(cmd, "/bin/bash")
	if err != nil {
		return "", err
	}
	// 移除所有换行符，保持与原有行为一致
	return strings.ReplaceAll(result, "\n", ""), nil
}

const (
	MegacliPath = "/opt/MegaRAID/MegaCli/MegaCli64"
	StorcliPath = "/opt/MegaRAID/storcli/storcli64"
	ArcconfPath = "/usr/sbin/arcconf"
	UnknownTool = "unknown"
)

var ToolMap = map[string]string{
	"LSI Logic / Symbios Logic MegaRAID SAS 2208":                    MegacliPath,
	"Broadcom / LSI MegaRAID SAS-3 3008":                             MegacliPath,
	"LSI Logic / Symbios Logic MegaRAID SAS-3 3008":                  MegacliPath,
	"LSI Logic / Symbios Logic MegaRAID SAS-3 3108":                  MegacliPath,
	"LSI Logic / Symbios Logic MegaRAID SAS-3 3316":                  MegacliPath,
	"Broadcom / LSI MegaRAID SAS-3 3316":                             MegacliPath,
	"LSI Logic / Symbios Logic MegaRAID SAS 2008":                    MegacliPath,
	"Broadcom / LSI MegaRAID SAS 2208":                               MegacliPath,
	"Broadcom / LSI MegaRAID SAS-3 3108":                             MegacliPath,
	"Broadcom / LSI SAS3008 PCI-Express Fusion-MPT SAS-3":            StorcliPath,
	"LSI Logic / Symbios Logic SAS3008 PCI-Express Fusion-MPT SAS-3": StorcliPath,
	"Broadcom / LSI MegaRAID Tri-Mode SAS3408":                       StorcliPath,
	"LSI Logic / Symbios Logic MegaRAID Tri-Mode SAS3408":            StorcliPath,
	"LSI Logic / Symbios Logic MegaRAID Tri-Mode SAS3508":            StorcliPath,
	"Broadcom / LSI MegaRAID Tri-Mode SAS3508":                       StorcliPath,
	"Broadcom / LSI MegaRAID 12GSAS/PCIe Secure SAS39xx":             StorcliPath,
	"Adaptec Series 8 12G SAS/PCIe 3":                                ArcconfPath,
	"Adaptec Smart Storage PQI SAS":                                  ArcconfPath,
	"Adaptec Device 028f":                                            ArcconfPath,
}

func ChooseTool(c string) string {
	if tool, exists := ToolMap[c]; exists {
		return tool
	}
	return UnknownTool
}

func checkTool(t string) bool {
	return PathExists(t)
}

func Collect() Controller {
	output, err := bash(`lspci | grep "^[0-9,a-z]" | grep -E 'Fusion-MPT|MegaRAID|Adaptec' | awk -F ':' '{print $NF}' | awk -F '[(|[]' '{print $1}' | uniq`)
	if err != nil {
		// 如果命令失败，返回未知工具
		return Controller{Name: "", Num: 0, Tool: UnknownTool, Avail: false}
	}
	c := strings.TrimSpace(output)

	t := ChooseTool(c)
	var cnum string
	switch t {
	case MegacliPath:
		cnumOutput, err := bash(fmt.Sprintf(`%s -adpCount -NoLog | grep Count | awk '{print $3}' | awk -F. '{print $1}'`, t))
		if err == nil {
			cnum = cnumOutput
		} else {
			cnum = "0"
		}
	case StorcliPath:
		cnumOutput, err := bash(fmt.Sprintf(`%s show | grep "Number of Controllers" | awk '{print $NF}'`, t))
		if err == nil {
			cnum = cnumOutput
		} else {
			cnum = "0"
		}
	case ArcconfPath:
		cnumOutput, err := bash(fmt.Sprintf(`%s list | grep "Controllers found:" | awk '{print $NF}'`, t))
		if err == nil {
			cnum = cnumOutput
		} else {
			cnum = "0"
		}
	default:
		cnum = "0"
	}
	cn, _ := strconv.Atoi(cnum)
	b := checkTool(t)
	// fmt.Println(b)
	s := Controller{Name: c, Num: cn, Tool: t, Avail: b}
	return s
}
