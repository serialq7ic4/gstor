package block

import (
	"bytes"
	"errors"
	"os/exec"

	"github.com/chenq7an/gstor/common/controller"
)

type Disk struct {
	Name         string `json:"name"`
	CES          string `json:"ces"`
	State        string `json:"state"`
	MediaType    string `json:"mediatype"`
	PDType       string `json:"pdtype"`
	MediaError   string `json:"mediaerror"`
	PredictError string `json:"predicterror"`
	Vendor       string `json:"vendor"`
	Capacity     string `json:"capcity"`
	SerialNumber string `json:"serialnumber"`
}

type DiskCollector interface {
	Collect() []Disk
	TurnOn(slot string) error
	TurnOff(slot string) error
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
		return ""
		// log.Fatalf("cmd.Run() failed with %s\n", cmd)
	}
	return outStr // strings.Split(strings.Trim(outStr, "\n"), "\n")
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
