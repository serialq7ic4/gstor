package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/chenq7an/gstor/common/block"
	"github.com/spf13/cobra"
)

type Report struct {
	Type    string   `json:"type"`
	IP      string   `json:"ip"`
	SN      string   `json:"sn"`
	Source  string   `json:"source"`
	Message []string `json:"message"`
}

var apiUrl string

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "数据上报",
	Long: `将硬盘故障按指定格式上报给硬件故障自助报修系统(Autobot->HWError)，数据格式形如：
{
	"type": "disk",
	"ip": "10.1.144.244",
	"sn": "9800077700659407",
	"source": "gstor",
	"message" : ["960 G_SSD_sdb_mediaerror_28",
	             "4 T_HDD_sdg_mediumerror_299"]
}`,
	Run: func(cmd *cobra.Command, args []string) {
		payload := Report{}
		var s []string
		disk, err := block.Devices()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		devices := disk.Collect()
		for _, v := range devices {
			if v.MediaError > "0" {
				s = append(s, v.Capacity+"_"+v.PDType+"_"+v.MediaType+"_"+v.Name+"_mediaerror_"+v.MediaError)
			} else if v.State == "Failed" || v.State == "Offline" {
				s = append(s, v.Capacity+"_"+v.PDType+"_"+v.MediaType+"_"+v.Name+"_"+v.State)
			}
		}
		payload.Type = "disk"
		payload.IP = bash(`route -n | grep ^[0-9] | grep -v docker | grep -v "169.254.0.0" | \
													awk '{print $NF}' | head -n1 | xargs -i ifconfig {} | grep inet | \
													grep netmask | grep broadcast | awk '{print $2}'`)
		payload.SN = bash(`dmidecode -s system-serial-number`)
		payload.Source = "gstor"
		payload.Message = s
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			fmt.Println("Json error:", err)
		}
		reader := bytes.NewReader(jsonPayload)
		request, _ := http.NewRequest("POST", apiUrl, reader)
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		client := &http.Client{}
		response, error := client.Do(request)
		if error != nil {
			panic(error)
		}
		defer response.Body.Close()
		fmt.Println("response Status:", response.Status)
		body, _ := ioutil.ReadAll(response.Body)
		fmt.Println("response Body:", string(body))
	},
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
		log.Fatalf("%s failed with %s\n", cmd, err)
	}
	return strings.Replace(outStr, "\n", "", -1)
}

func init() {
	rootCmd.AddCommand(reportCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// reportCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// reportCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	reportCmd.Flags().StringVarP(&apiUrl, "url", "u", "", "指定数据上报的api")
	reportCmd.MarkFlagRequired("url")
}
