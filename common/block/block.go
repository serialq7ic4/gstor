package block

import (
	"bytes"
	"errors"
	"gstor/common/controller"
	"log"
	"os/exec"
)

type Disk struct {
	Name         string
	CES          string
	State        string
	MediaType    string
	MediaError   string
	PredictError string
	TargetId     string

	Vendor       string
	Capacity     string
	SerialNumber string
}

// 内存逃逸?
type DiskCollector interface {
	Collect() []*Disk
}

func Bash(cmd string) string {
	cmdjob := exec.Command("/bin/sh", "-c", cmd)
	var stdout, stderr bytes.Buffer
	cmdjob.Stdout = &stdout
	cmdjob.Stderr = &stderr
	err := cmdjob.Run()
	outStr, _ := stdout.String(), stderr.String()
	// fmt.Printf("out:%serr:%s\n", outStr, errStr)
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", cmd)
	}
	return outStr //strings.Split(strings.Trim(outStr, "\n"), "\n")
}

func Devices() (DiskCollector, error) {
	c := controller.Collect()

	switch c.Tool {
	case "/opt/MegaRAID/MegaCli/MegaCli64":
		return &megacliCollector{}, nil
	case "/opt/MegaRAID/storcli/storcli64":
		return &storcliCollector{}, nil
	case "/usr/sbin/arcconf":
		return &arcconfCollector{}, nil
	default:
		return nil, errors.New("unknown raid tool")
	}
}
