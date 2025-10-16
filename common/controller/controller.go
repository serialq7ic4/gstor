package controller

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Crontroller struct {
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

func bash(cmd string) string {
	cmdjob := exec.Command("/bin/bash", "-c", cmd)
	var stdout, stderr bytes.Buffer
	cmdjob.Stdout = &stdout
	cmdjob.Stderr = &stderr
	err := cmdjob.Run()
	outStr, _ := stdout.String(), stderr.String()
	// fmt.Printf("out:%serr:%s\n", outStr, errStr)
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	return strings.Replace(outStr, "\n", "", -1)
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

func Collect() Crontroller {
	output := bash(`lspci | grep "^[0-9,a-z]" | grep -E 'Fusion-MPT|MegaRAID|Adaptec' | awk -F ':' '{print $NF}' | awk -F '[(|[]' '{print $1}' | uniq`)
	c := strings.TrimSpace(output)

	t := ChooseTool(c)
	var cnum string
	switch t {
	case MegacliPath:
		cnum = bash(fmt.Sprintf(`%s -adpCount -NoLog | grep Count | awk '{print $3}' | awk -F. '{print $1}'`, t))
	case StorcliPath:
		cnum = bash(fmt.Sprintf(`%s show | grep "Number of Controllers" | awk '{print $NF}'`, t))
	case ArcconfPath:
		cnum = bash(fmt.Sprintf(`%s list | grep "Controllers found:" | awk '{print $NF}'`, t))
	default:
		cnum = "0"
	}
	cn, _ := strconv.Atoi(cnum)
	b := checkTool(t)
	// fmt.Println(b)
	s := Crontroller{Name: c, Num: cn, Tool: t, Avail: b}
	return s
}
