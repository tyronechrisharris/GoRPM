package simulator

import (
	"fmt"
	"math"
	"strings"
)

type DetectorValues struct {
	MsgType    string
	TimeOffset float64
	Values     []int
	MsgTime    float64
}

func (dv *DetectorValues) TotalCounts() int {
	sum := 0
	for _, v := range dv.Values {
		sum += v
	}
	return sum
}

func (dv *DetectorValues) String() string {
	if dv.MsgType == "SP" {
		return "SP,0.2249,03.032,004.88,000000"
	}

	vals := make([]string, len(dv.Values))
	for i, v := range dv.Values {
		vals[i] = fmt.Sprintf("%06d", v)
	}
	return fmt.Sprintf("%s,%s", dv.MsgType, strings.Join(vals, ","))
}

type RPMProfile struct {
	Counts []DetectorValues
	Type   string
	Cursor int
}

func NewRPMProfile() *RPMProfile {
	return &RPMProfile{
		Counts: make([]DetectorValues, 0),
		Type:   "oc", // default
		Cursor: 0,
	}
}

func (p *RPMProfile) AddSample(vals DetectorValues) {
	p.Counts = append(p.Counts, vals)
	if vals.MsgType == "GA" {
		if p.Type == "na" || p.Type == "ng" {
			p.Type = "ng"
		} else {
			p.Type = "ga"
		}
	} else if vals.MsgType == "NA" {
		if p.Type == "ga" || p.Type == "ng" {
			p.Type = "ng"
		} else {
			p.Type = "na"
		}
	}
}

func (p *RPMProfile) AddGX(counter int) {
	if len(p.Counts) > 0 {
		lastMsgTime := p.Counts[len(p.Counts)-1].MsgTime
		gxVals := DetectorValues{
			MsgType: "GX",
			TimeOffset: lastMsgTime,
			Values:  []int{counter, counter * 10, 0, 0},
		}
		p.AddSample(gxVals)
	}
}

func (p *RPMProfile) AddTimeOffset(offset float64) {
	for i := range p.Counts {
		p.Counts[i].MsgTime = offset + (p.Counts[i].TimeOffset / 1000.0)
	}
	p.Cursor = 0
}

func (p *RPMProfile) IsEOF() bool {
	return p.Cursor >= len(p.Counts)
}

func (p *RPMProfile) GetNextMessage(now float64) *DetectorValues {
	if p.IsEOF() {
		return nil
	}
	dv := &p.Counts[p.Cursor]
	if now >= dv.MsgTime {
		p.Cursor++
		return dv
	}
	return nil
}

type ProfileGenerator struct{}

func normPDF(x, mean, stddev float64) float64 {
	variance := stddev * stddev
	return math.Exp(-math.Pow(x-mean, 2)/(2*variance)) / math.Sqrt(2*math.Pi*variance)
}

func (pg *ProfileGenerator) GenerateProfile(params map[string]float64) [][2]float64 {
	duration := params["duration"]
	if duration == 0 {
		duration = 10.0
	}
	stddev := params["stddev"]
	if stddev == 0 {
		stddev = duration / 2.0
	}
	timeInc := params["time_increment"]
	if timeInc == 0 {
		timeInc = 1
	}
	humps := int(params["humps"])
	if humps == 0 {
		humps = 1
	}
	shift := params["shift"]

	numPoints := int(duration * 5 / timeInc)
	xValues := make([]float64, numPoints)
	for i := 0; i < numPoints; i++ {
		xValues[i] = float64(i) * timeInc
	}

	mean := (xValues[0] + xValues[len(xValues)-1]) / 2.0
	yValues := make([]float64, numPoints)

	maxVal := 0.0

	if humps > 1 {
		mean1 := mean * (1.0 - shift)
		mean2 := mean * (1.0 + shift)
		for i, x := range xValues {
			val1 := normPDF(x, mean1, stddev)
			val2 := normPDF(x, mean2, stddev)
			yValues[i] = val1 + val2
			if yValues[i] > maxVal {
				maxVal = yValues[i]
			}
		}
	} else {
		shiftedMean := mean * (1.0 + shift)
		for i, x := range xValues {
			yValues[i] = normPDF(x, shiftedMean, stddev)
			if yValues[i] > maxVal {
				maxVal = yValues[i]
			}
		}
	}

	if maxVal > 0 {
		for i := range yValues {
			yValues[i] /= maxVal
		}
	}

	result := make([][2]float64, numPoints)
	for i := range xValues {
		result[i] = [2]float64{xValues[i], yValues[i]}
	}

	return result
}
