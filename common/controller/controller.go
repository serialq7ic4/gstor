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
	cmdjob := exec.Command("/bin/sh", "-c", cmd)
	var stdout, stderr bytes.Buffer
	cmdjob.Stdout = &stdout
	cmdjob.Stderr = &stderr
	err := cmdjob.Run()
	outStr, _ := string(stdout.Bytes()), string(stderr.Bytes())
	// fmt.Printf("out:%serr:%s\n", outStr, errStr)
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	return strings.Replace(outStr, "\n", "", -1)
}

func ChooseTool(c string) string {
	var t string
	switch c {
	case "LSI Logic / Symbios Logic MegaRAID SAS 2208",
		"LSI Logic / Symbios Logic MegaRAID SAS-3 3008",
		"LSI Logic / Symbios Logic MegaRAID SAS-3 3108",
		"LSI Logic / Symbios Logic MegaRAID SAS-3 3316",
		"LSI Logic / Symbios Logic MegaRAID SAS 2008",
		"Broadcom / LSI MegaRAID SAS 2208":
		t = `/opt/MegaRAID/MegaCli/MegaCli64`
	case "LSI Logic / Symbios Logic SAS3008 PCI-Express Fusion-MPT SAS-3",
		"LSI Logic / Symbios Logic MegaRAID Tri-Mode SAS3408",
		"Broadcom / LSI SAS3008 PCI-Express Fusion-MPT SAS-3":
		t = `/opt/MegaRAID/storcli/storcli64`
	case "Adaptec Series 8 12G SAS/PCIe 3",
		"Adaptec Device 028f":
		t = `/usr/sbin/arcconf`
	default:
		t = "unknown"
	}
	return t
}

func checkTool(t string) bool {
	return PathExists(t)
}

func Collect() Crontroller {
	c := bash(`lspci | grep "^[0-9,a-z]" | grep -E 'Fusion-MPT|MegaRAID|Adaptec' | awk -F ':' '{print $NF}' | awk -F '[(|[]' '{print $1}' | awk '{gsub(/^\s+|\s+$/, "");print}' | uniq`)
	t := ChooseTool(c)
	var cnum string
	switch t {
	case "/opt/MegaRAID/MegaCli/MegaCli64":
		cnum = bash(fmt.Sprintf(`%s -adpCount | grep Count | awk '{print $3}' | awk -F. '{print $1}'`, t))
	case "/opt/MegaRAID/storcli/storcli64":
		cnum = bash(fmt.Sprintf(`%s show | grep "Number of Controllers" | awk '{print $NF}'`, t))
	case "/usr/sbin/arcconf":
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
