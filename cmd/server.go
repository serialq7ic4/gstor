package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chenq7an/gstor/common/block"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

func getCurrentPath() string {
	if ex, err := os.Executable(); err == nil {
		return filepath.Dir(ex)
	}
	return "./"
}

// serverCmd represents the server command
var (
	port      string
	f         embed.FS
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "启动http服务,使用方法: gstor server --port=?",
		Run: func(cmd *cobra.Command, args []string) {
			if port == "" {
				fmt.Println("port不能为空")
				os.Exit(1)
			}
			r := gin.Default()
			// r.LoadHTMLFiles("./templates/dashboard.tmpl")
			r.GET("/", func(c *gin.Context) {
				tmpl := template.New("")
				resp, err := http.Get("https://oss-beijing-m8.openstorage.cn/cephstuff/dashboard.tmpl")
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to load remote HTML file")
					return
				}
				defer resp.Body.Close()

				bodyBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to read remote HTML file")
					return
				}

				// 将字节数组转换为字符串类型
				bodyString := string(bodyBytes)

				_, err = tmpl.Parse(bodyString)
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to parse remote HTML file")
					return
				}

				var disk []block.Disk
				data := showBlock("json")
				if err := json.Unmarshal([]byte(data), &disk); err != nil {
					fmt.Println("解析 JSON 失败：", err)
					return
				}
				err = tmpl.Execute(c.Writer, data)
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to render remote HTML file")
					return
				}
				// c.HTML(http.StatusOK, "dashboard.tmpl", gin.H{"disks": disk})
			})
			r.GET("/disks", func(c *gin.Context) {
				ret := showBlock("json")
				c.JSON(200, gin.H{"disks": ret})
			})
			_ = r.Run(":" + port)
		},
	}
)

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	serverCmd.Flags().StringVar(&port, "port", "", "端口号")
}
