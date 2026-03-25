package simulator

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/sandialabs/srls-go/internal/config"
)

type LaneSimulatorIface interface {
	GenerateAlarm(alarmType string, durationS float64)
}

type RPMSimulator struct {
	owner              LaneSimulatorIface
	settings           config.RPMSettings
	autoGammaProb      float64
	autoNeutronProb    float64
	autoIntervalS      int
	name               string

	profileGen         ProfileGenerator
	queuedProfiles     []*RPMProfile
	currentProfile     *RPMProfile
	gxCounter          int
	nextBgTime         float64
	bgIntervalS        float64
	autoModeActive     bool
	autoNextOccTime    float64

	server             *TCPServer
	quitChan           chan struct{}
	running            bool
	mux                sync.Mutex
}

func NewRPMSimulator(name string, laneSettings config.LaneSettings, owner LaneSimulatorIface) *RPMSimulator {
	return &RPMSimulator{
		owner:           owner,
		settings:        laneSettings.RPM,
		autoGammaProb:   laneSettings.AutoGammaProbability,
		autoNeutronProb: laneSettings.AutoNeutronProbability,
		autoIntervalS:   laneSettings.AutoInterval,
		name:            name,
		bgIntervalS:     5.0,
		queuedProfiles:  make([]*RPMProfile, 0),
		quitChan:        make(chan struct{}),
	}
}

func (r *RPMSimulator) Start() {
	if r.running {
		return
	}
	r.running = true
	addr := fmt.Sprintf("%s:%d", r.settings.IPAddr, r.settings.Port)
	r.server = NewTCPServer(addr)
	if err := r.server.Start(); err != nil {
		log.Printf("[%s] failed to start TCP server on %s: %v", r.name, addr, err)
		return
	}

	log.Printf("[%s] Started RPM simulator on %s", r.name, addr)

	r.nextBgTime = float64(time.Now().UnixNano()) / 1e9

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-r.quitChan:
				return
			case <-ticker.C:
				r.tick()
			}
		}
	}()
}

func (r *RPMSimulator) Stop() {
	if !r.running {
		return
	}
	r.running = false
	close(r.quitChan)
	if r.server != nil {
		r.server.Stop()
	}
	log.Printf("[%s] Stopped RPM simulator", r.name)
}

func (r *RPMSimulator) StartAutoMode() {
	r.mux.Lock()
	defer r.mux.Unlock()
	if !r.autoModeActive {
		r.autoNextOccTime = float64(time.Now().UnixNano()) / 1e9
		r.autoModeActive = true
		log.Printf("[%s] Auto mode started", r.name)
	}
}

func (r *RPMSimulator) StopAutoMode() {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.autoModeActive {
		r.autoModeActive = false
		log.Printf("[%s] Auto mode stopped", r.name)
	}
}

func (r *RPMSimulator) AutoModeActive() bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.autoModeActive
}

func (r *RPMSimulator) OccupancyState() string {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.currentProfile != nil {
		return r.currentProfile.Type
	}
	return "unoccupied"
}

func (r *RPMSimulator) ClientCount() int {
	if r.server != nil {
		return r.server.ClientCount()
	}
	return 0
}

func distributeCounts(total float64, distribution []float64) []int {
	counts := make([]int, len(distribution))
	for i, w := range distribution {
		counts[i] = int(math.Round(total * w))
	}
	if len(counts) > 1 && rand.Float64() > 0.5 {
		idx1 := rand.Intn(len(counts))
		idx2 := rand.Intn(len(counts))
		for idx1 == idx2 {
			idx2 = rand.Intn(len(counts))
		}
		if counts[idx1] > 0 {
			counts[idx1]--
			counts[idx2]++
		}
	}
	for i := range counts {
		if counts[i] < 0 {
			counts[i] = 0
		}
	}
	return counts
}

func (r *RPMSimulator) GenerateFromModel(model map[string]float64) {
	log.Printf("[%s] Generating profile from model", r.name)

	gammaParams := map[string]float64{
		"duration":       model["duration"],
		"stddev":         model["stddev"],
		"time_increment": 1.0,
		"humps":          model["humps"],
		"shift":          model["shift"],
	}
	gammaCurve := r.profileGen.GenerateProfile(gammaParams)

	neutronParams := map[string]float64{
		"duration":       model["duration"],
		"stddev":         model["stddev"],
		"time_increment": 5.0,
		"humps":          model["humps"],
		"shift":          model["shift"],
	}
	neutronCurve := r.profileGen.GenerateProfile(neutronParams)

	gammaBgS := r.settings.GammaBG
	sqrtGammaBg := math.Sqrt(gammaBgS)

	maxGammaCount200ms := (model["gamma_nsigma"]*sqrtGammaBg + gammaBgS) / 5.0
	gammaOffset := maxGammaCount200ms - (gammaBgS / 5.0)

	gammaAlarmThreshold1s := r.settings.GHThreshold
	neutronAlarmThreshold1s := r.settings.NeutronThreshold

	profile := NewRPMProfile()

	gIdx := 0
	nIdx := 0

	for gIdx < len(gammaCurve) || nIdx < len(neutronCurve) {
		takeGamma := false
		if gIdx < len(gammaCurve) && nIdx < len(neutronCurve) {
			if gammaCurve[gIdx][0] <= neutronCurve[nIdx][0] {
				takeGamma = true
			}
		} else if gIdx < len(gammaCurve) {
			takeGamma = true
		}

		if takeGamma {
			timeIdx := gammaCurve[gIdx][0]
			amplitude := gammaCurve[gIdx][1]
			gIdx++

			totalCount200ms := (gammaBgS / 5.0) + (amplitude * gammaOffset)
			counts := distributeCounts(totalCount200ms, r.settings.GammaDistribution)

			isAlarm := totalCount200ms*5 > gammaAlarmThreshold1s
			msgType := "GS"
			if isAlarm {
				msgType = "GA"
			}

			dv := DetectorValues{MsgType: msgType, TimeOffset: timeIdx * 200, Values: counts}
			profile.AddSample(dv)

			if len(profile.Counts)%5 == 0 {
				profile.AddSample(DetectorValues{MsgType: "SP", TimeOffset: timeIdx * 200, Values: []int{0, 0, 0, 0}})
			}
		} else {
			timeIdx := neutronCurve[nIdx][0]
			amplitude := neutronCurve[nIdx][1]
			nIdx++

			totalCount1s := r.settings.NeutronBG + (amplitude * model["neutron_amplitude"])
			counts := distributeCounts(totalCount1s, r.settings.NeutronDistribution)

			isAlarm := totalCount1s > neutronAlarmThreshold1s
			msgType := "NS"
			if isAlarm {
				msgType = "NA"
			}
			dv := DetectorValues{MsgType: msgType, TimeOffset: timeIdx * 200, Values: counts}
			profile.AddSample(dv)
		}
	}

	r.gxCounter++
	profile.AddGX(r.gxCounter)

	r.mux.Lock()
	r.queuedProfiles = append(r.queuedProfiles, profile)
	r.mux.Unlock()
}

func (r *RPMSimulator) tick() {
	now := float64(time.Now().UnixNano()) / 1e9

	r.mux.Lock()
	if r.currentProfile == nil && len(r.queuedProfiles) > 0 {
		r.currentProfile = r.queuedProfiles[0]
		r.queuedProfiles = r.queuedProfiles[1:]
		r.currentProfile.AddTimeOffset(now)
		log.Printf("[%s] Starting new profile of type: %s", r.name, r.currentProfile.Type)
	}

	if r.currentProfile != nil {
		msg := r.currentProfile.GetNextMessage(now)
		msgsToSend := ""
		for msg != nil {
			msgsToSend += msg.String() + "\r\n"
			msg = r.currentProfile.GetNextMessage(now)
		}

		if msgsToSend != "" {
			r.server.Broadcast(msgsToSend)
		}

		if r.currentProfile.IsEOF() {
			log.Printf("[%s] Profile finished.", r.name)
			r.currentProfile = nil
			r.nextBgTime = now + r.bgIntervalS
			if r.autoModeActive {
				r.autoNextOccTime = now + float64(r.autoIntervalS)
			}
		}
	} else {
		if r.autoModeActive && now >= r.autoNextOccTime {
			r.triggerAutoOccupancy()
			r.autoNextOccTime = now + 99999.0
		} else if now >= r.nextBgTime {
			r.sendBackgroundCounts()
			r.nextBgTime = now + r.bgIntervalS
		}
	}
	r.mux.Unlock()
}

func (r *RPMSimulator) triggerAutoOccupancy() {
	log.Printf("[%s] Auto-mode triggering new occupancy.", r.name)
	isGamma := rand.Float64() <= r.autoGammaProb
	isNeutron := rand.Float64() <= r.autoNeutronProb

	alarmType := "OC"
	if isGamma && isNeutron {
		alarmType = "NG"
	} else if isGamma {
		alarmType = "GA"
	} else if isNeutron {
		alarmType = "NA"
	}

	go r.owner.GenerateAlarm(alarmType, -1.0)
}

func (r *RPMSimulator) sendBackgroundCounts() {
	if r.ClientCount() == 0 {
		return
	}

	nCounts := distributeCounts(r.settings.NeutronBG, r.settings.NeutronDistribution)
	nTotal := 0
	for _, c := range nCounts {
		nTotal += c
	}
	nMsgType := "NB"
	if float64(nTotal) > r.settings.NHThreshold {
		nMsgType = "NH"
	}
	nDv := DetectorValues{MsgType: nMsgType, TimeOffset: 0, Values: nCounts}

	gCounts := distributeCounts(r.settings.GammaBG, r.settings.GammaDistribution)
	gTotal := 0
	for _, c := range gCounts {
		gTotal += c
	}
	gMsgType := "GB"
	if float64(gTotal) > r.settings.GHThreshold {
		gMsgType = "GH"
	} else if float64(gTotal) < r.settings.GLThreshold {
		gMsgType = "GL"
	}
	gDv := DetectorValues{MsgType: gMsgType, TimeOffset: 0, Values: gCounts}

	r.server.Broadcast(fmt.Sprintf("%s\r\n%s\r\n", nDv.String(), gDv.String()))
}
