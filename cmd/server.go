package cmd

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"text/template"

	"github.com/chenq7an/gstor/common/block"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start web server to display command results",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		startServer(port)
	},
}

func init() {
	serverCmd.Flags().IntP("port", "p", 9100, "Port to listen on")
	rootCmd.AddCommand(serverCmd)
}

func startServer(port int) {
	http.HandleFunc("/disks", func(w http.ResponseWriter, r *http.Request) {
		data := getCommandResults()
		renderTable(w, data)
	})

	http.HandleFunc("/disks/locate/on/", func(w http.ResponseWriter, r *http.Request) {
		slot := r.URL.Path[len("/locate/on/"):]
		disk, err := block.Devices()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = disk.TurnOn(slot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/disks/locate/off/", func(w http.ResponseWriter, r *http.Request) {
		slot := r.URL.Path[len("/locate/off/"):]
		disk, err := block.Devices()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = disk.TurnOff(slot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("OK"))
	})

	// Create reverse proxy to handle requests
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = req.Host
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/disks", func(w http.ResponseWriter, r *http.Request) {
		data := getCommandResults()
		renderTable(w, data)
	})
	mux.HandleFunc("/disks/locate/on/", func(w http.ResponseWriter, r *http.Request) {
		slot := r.URL.Path[len("/disks/locate/on/"):]
		disk, err := block.Devices()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = disk.TurnOn(slot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/disks/locate/off/", func(w http.ResponseWriter, r *http.Request) {
		slot := r.URL.Path[len("/disks/locate/off/"):]
		disk, err := block.Devices()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = disk.TurnOff(slot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("OK"))
	})

	// Handle all other requests with reverse proxy
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: mux,
	}

	fmt.Printf("Starting server at http://%s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
}

func getCommandResults() []map[string]string {
	disk, err := block.Devices()
	if err != nil {
		return []map[string]string{
			{"Error": err.Error()},
		}
	}

	devices := disk.Collect()
	var results []map[string]string

	for _, device := range devices {
		results = append(results, map[string]string{
			"Disk":         device.Name,
			"SN":           device.SerialNumber,
			"Capacity":     device.Capacity,
			"Vendor":       device.Vendor,
			"Model":        device.Model,
			"PDType":       device.PDType,
			"MediaType":    device.MediaType,
			"Slot":         device.CES,
			"State":        device.State,
			"MediaError":   fmt.Sprintf("%v", device.MediaError),
			"PredictError": fmt.Sprintf("%v", device.PredictError),
		})
	}

	return results
}

func renderTable(w http.ResponseWriter, data []map[string]string) {
	const tpl = `
<!DOCTYPE html>
<html>
<head>
    <title>Command Results</title>
    <style>
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        tr:hover {background-color:#f5f5f5;}
        .error-row {background-color: orange;}
    </style>
</head>
<body>
    <h1>Command Results</h1>
    <table>
        <tr>
            <th>Disk</th>
            <th>SN</th>
            <th>Capacity</th>
            <th>Vendor</th>
            <th>Model</th>
            <th>PDType</th>
            <th>MediaType</th>
            <th>Slot</th>
            <th>State</th>
            <th>MediaError</th>
            <th>PredictError</th>
            <th>Locate On</th>
            <th>Locate Off</th>
        </tr>
        {{range .}}<tr {{if or (ne .MediaError "0") (ne .PredictError "0")}}class="error-row"{{end}}>
            <td>{{.Disk}}</td>
            <td>{{.SN}}</td>
            <td>{{.Capacity}}</td>
            <td>{{.Vendor}}</td>
            <td>{{.Model}}</td>
            <td>{{.PDType}}</td>
            <td>{{.MediaType}}</td>
            <td>{{.Slot}}</td>
            <td>{{.State}}</td>
            <td>{{.MediaError}}</td>
            <td>{{.PredictError}}</td>
            <td>
                <button onclick="locateOn('{{.Slot}}')">Locate On</button>
            </td>
            <td>
                <button onclick="locateOff('{{.Slot}}')">Locate Off</button>
            </td>
        </tr>{{end}}
    </table>
    <script>
        function locateOn(slot) {
            fetch('/disks/locate/on/' + slot)
                .then(response => response.text())
                .then(data => alert(data))
                .catch(error => alert('Error: ' + error));
        }
        function locateOff(slot) {
            fetch('/disks/locate/off/' + slot)
                .then(response => response.text())
                .then(data => alert(data))
                .catch(error => alert('Error: ' + error));
        }
    </script>
</body>
</html>`

	tmpl, err := template.New("webpage").Parse(tpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
