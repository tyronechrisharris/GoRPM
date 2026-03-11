package simulator

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/sandialabs/srls-go/internal/config"
	"github.com/sandialabs/srls-go/internal/rtsp"
)

type LaneSimulator struct {
	Settings   config.LaneSettings
	Name       string
	IsEnabled  bool
	rpm        *RPMSimulator
	rtspServer *rtsp.Server
}

func NewLaneSimulator(settings config.LaneSettings) *LaneSimulator {
	ls := &LaneSimulator{
		Settings:  settings,
		Name:      settings.LaneName,
		IsEnabled: settings.Enabled,
	}

	ls.rpm = NewRPMSimulator(fmt.Sprintf("%s-RPM", ls.Name), settings, ls)

	portNum := 8554
	if settings.LaneID != 1 {
		portNum = 8554 + settings.LaneID - 1
	}

	ls.rtspServer = rtsp.NewServer(fmt.Sprintf("%d", portNum))

	log.Printf("Lane '%s' initialized.", ls.Name)
	return ls
}

func (l *LaneSimulator) Start() {
	if l.IsEnabled {
		log.Printf("[%s] Starting lane...", l.Name)
		l.rpm.Start()
		l.rtspServer.Start()
	} else {
		log.Printf("[%s] Lane is disabled, not starting.", l.Name)
	}
}

func (l *LaneSimulator) Stop() {
	log.Printf("[%s] Stopping lane...", l.Name)
	if l.rpm != nil {
		l.rpm.Stop()
	}
	if l.rtspServer != nil {
		l.rtspServer.Stop()
	}
}

func (l *LaneSimulator) PollStatus() (string, int, string) {
	status := "stopped"
	if l.IsEnabled {
		status = "running"
	} else {
		status = "disabled"
	}

	clientCount := l.rpm.ClientCount()
	occupancyState := l.rpm.OccupancyState()

	if l.rtspServer != nil {
		l.rtspServer.SetOccupied(occupancyState != "unoccupied")
	}

	return status, clientCount, occupancyState
}

func (l *LaneSimulator) SetAutoMode(autoOn bool) {
	if l.IsEnabled {
		if autoOn {
			l.rpm.StartAutoMode()
		} else {
			l.rpm.StopAutoMode()
		}
	}
}

func (l *LaneSimulator) GenerateAlarm(alarmType string, durationS float64) {
	if !l.IsEnabled {
		return
	}

	log.Printf("[%s] Generating '%s' alarm.", l.Name, alarmType)

	if durationS <= 0 {
		durationS = 7.0 + rand.Float64()*10.0 // 7-17 seconds
	}

	gammaNsigma := 0.0
	if alarmType == "GA" || alarmType == "NG" {
		gammaNsigma = l.rpm.settings.GammaNSigma + rand.Float64()*2.0
	}

	neutronAmplitude := 0.0
	if alarmType == "NA" || alarmType == "NG" {
		neutronAmplitude = l.rpm.settings.NeutronThreshold + rand.Float64()*3.0
	}

	humps := 1
	if rand.Float64() < 0.7 {
		humps = 1
	} else {
		humps = 2
	}

	shift := (0.5 - rand.Float64()) * 0.8

	model := map[string]float64{
		"duration":          durationS,
		"stddev":            rand.Float64()*durationS*0.5 + 2.0,
		"humps":             float64(humps),
		"shift":             shift,
		"gamma_nsigma":      gammaNsigma,
		"neutron_amplitude": neutronAmplitude,
	}

	l.rpm.GenerateFromModel(model)
}
