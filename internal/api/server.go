package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/sandialabs/srls-go/internal/simulator"
)

type Server struct {
	port  int
	lanes []*simulator.LaneSimulator
}

func StartServer(port int, lanes []*simulator.LaneSimulator) {
	server := &Server{
		port:  port,
		lanes: lanes,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleIndex)
	mux.HandleFunc("/api/status", server.handleStatus)
	mux.HandleFunc("/api/alarm", server.handleAlarm)
	mux.HandleFunc("/api/auto", server.handleAuto)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting web server on http://127.0.0.1%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("Web server error: %v", err)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, indexHTML)
}

type LaneStatus struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Clients   int    `json:"clients"`
	Occupancy string `json:"occupancy"`
	AutoMode  bool   `json:"auto_mode"`
	IPAddr    string `json:"ip_addr"`
	RPMPort   int    `json:"rpm_port"`
	VideoPort int    `json:"video_port"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var statuses []LaneStatus
	for _, lane := range s.lanes {
		status, clients, occupancy, autoMode := lane.PollStatus()
		videoPort := 8554
		if lane.Settings.LaneID != 1 {
			videoPort = 8554 + lane.Settings.LaneID - 1
		}
		statuses = append(statuses, LaneStatus{
			ID:        lane.GetID(),
			Name:      lane.Name,
			Status:    status,
			Clients:   clients,
			Occupancy: occupancy,
			AutoMode:  autoMode,
			IPAddr:    lane.Settings.RPM.IPAddr,
			RPMPort:   lane.Settings.RPM.Port,
			VideoPort: videoPort,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

type AlarmRequest struct {
	LaneID    int    `json:"lane_id"`
	AlarmType string `json:"alarm_type"`
}

func (s *Server) handleAlarm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AlarmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for _, lane := range s.lanes {
		if lane.GetID() == req.LaneID {
			lane.GenerateAlarm(req.AlarmType, -1.0)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	http.Error(w, "Lane not found", http.StatusNotFound)
}

type AutoRequest struct {
	LaneID int  `json:"lane_id"`
	AutoOn bool `json:"auto_on"`
}

func (s *Server) handleAuto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AutoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for _, lane := range s.lanes {
		if lane.GetID() == req.LaneID {
			lane.SetAutoMode(req.AutoOn)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	http.Error(w, "Lane not found", http.StatusNotFound)
}

var indexHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SRLS Simulator UI</title>
    <style>
        body { font-family: sans-serif; margin: 20px; }
        table { border-collapse: collapse; width: 100%; margin-bottom: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        button { margin-right: 5px; padding: 5px 10px; cursor: pointer; }
        .occupancy-GA { background-color: #ffcccc; }
        .occupancy-NA { background-color: #ccddff; }
        .occupancy-NG { background-color: #eeb8ff; }
        .occupancy-OC { background-color: #ffffcc; }
        .occupancy-unoccupied { background-color: #ccffcc; }
    </style>
</head>
<body>
    <h1>SRLS Simulator Control</h1>

    <table id="lanes-table">
        <thead>
            <tr>
                <th>Lane ID</th>
                <th>Name</th>
                <th>IP Address</th>
                <th>RPM Port</th>
                <th>Video Port</th>
                <th>Status</th>
                <th>Clients</th>
                <th>Occupancy</th>
                <th>Auto Mode</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            <!-- Populated via JS -->
        </tbody>
    </table>

    <script>
        async function fetchStatus() {
            try {
                const res = await fetch('/api/status');
                const data = await res.json();
                renderTable(data);
            } catch (err) {
                console.error("Error fetching status:", err);
            }
        }

        function renderTable(lanes) {
            const tbody = document.querySelector('#lanes-table tbody');
            tbody.innerHTML = '';

            lanes.forEach(lane => {
                const tr = document.createElement('tr');

                const occClass = 'occupancy-' + lane.occupancy;

                tr.innerHTML = '<td>' + lane.id + '</td>' +
                    '<td>' + lane.name + '</td>' +
                    '<td>' + lane.ip_addr + '</td>' +
                    '<td>' + lane.rpm_port + '</td>' +
                    '<td>' + lane.video_port + '</td>' +
                    '<td>' + lane.status + '</td>' +
                    '<td>' + lane.clients + '</td>' +
                    '<td class="' + occClass + '">' + lane.occupancy + '</td>' +
                    '<td>' + (lane.auto_mode ? 'ON' : 'OFF') + '</td>' +
                    '<td>' +
                        '<button onclick="toggleAuto(' + lane.id + ', ' + !lane.auto_mode + ')">Toggle Auto</button>' +
                        '<button onclick="injectAlarm(' + lane.id + ', \'GA\')">Gamma (GA)</button>' +
                        '<button onclick="injectAlarm(' + lane.id + ', \'NA\')">Neutron (NA)</button>' +
                        '<button onclick="injectAlarm(' + lane.id + ', \'NG\')">G+N (NG)</button>' +
                        '<button onclick="injectAlarm(' + lane.id + ', \'OC\')">Normal (OC)</button>' +
                    '</td>';
                tbody.appendChild(tr);
            });
        }

        async function toggleAuto(laneId, autoOn) {
            try {
                await fetch('/api/auto', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ lane_id: laneId, auto_on: autoOn })
                });
                fetchStatus();
            } catch (err) {
                console.error("Error toggling auto:", err);
            }
        }

        async function injectAlarm(laneId, type) {
            try {
                await fetch('/api/alarm', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ lane_id: laneId, alarm_type: type })
                });
                fetchStatus();
            } catch (err) {
                console.error("Error injecting alarm:", err);
            }
        }

        setInterval(fetchStatus, 1000);
        fetchStatus();
    </script>
</body>
</html>
`
