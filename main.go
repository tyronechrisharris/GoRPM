package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sandialabs/srls-go/internal/api"
	"github.com/sandialabs/srls-go/internal/config"
	"github.com/sandialabs/srls-go/internal/simulator"
)

func main() {
	fmt.Println("--- Sandia Radiation Portal Monitor Simulator (Go) ---")

	cfg, err := config.LoadConfig("settings.json")
	if err != nil {
		log.Fatalf("Failed to load settings: %v", err)
	}

	if cfg.LogFilename != "" {
		f, err := os.OpenFile(cfg.LogFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	var lanes []*simulator.LaneSimulator
	for _, laneConfig := range cfg.Lanes {
		if laneConfig.Enabled {
			lane := simulator.NewLaneSimulator(laneConfig)
			lanes = append(lanes, lane)
		}
	}

	if len(lanes) == 0 {
		log.Fatal("No enabled lanes found in settings.json. Exiting.")
	}

	for _, lane := range lanes {
		lane.Start()
		lane.SetAutoMode(true)
	}

	go api.StartServer(cfg.WebPort, lanes)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigs:
			fmt.Println("\nShutdown signal received. Stopping simulators...")
			for _, lane := range lanes {
				lane.Stop()
			}
			log.Println("All simulators stopped. Goodbye.")
			fmt.Println("Simulation finished.")
			return
		case <-ticker.C:
			fmt.Println("\n========================================")
			fmt.Printf("Status at %s\n", time.Now().Format("2006-01-02 15:04:05"))
			fmt.Println("========================================")
			for _, lane := range lanes {
				status, clients, occupancy, autoMode := lane.PollStatus()
				autoStr := "OFF"
				if autoMode {
					autoStr = "ON"
				}
				fmt.Printf("  Lane: %-15s | Status: %-10s | Clients: %-3d | Occupancy: %-12s | Auto: %-3s\n",
					lane.Name, status, clients, occupancy, autoStr)
			}
			fmt.Println("========================================")
			fmt.Println("(Press Ctrl+C to stop the simulator)")
		}
	}
}
