package config

import (
	"log"

	"github.com/spf13/viper"
)

type RPMSettings struct {
	IPAddr            string
	Port              int
	GammaBG           float64
	NeutronBG         float64
	GammaNSigma       float64
	NeutronThreshold  float64
	GHThreshold       float64
	GLThreshold       float64
	NHThreshold       float64
	GammaDistribution []float64
	NeutronDistribution []float64
}

type LaneSettings struct {
	LaneID                 int
	LaneName               string
	Enabled                bool
	AutoGammaProbability   float64
	AutoNeutronProbability float64
	AutoInterval           int
	RPM                    RPMSettings
}

type AppSettings struct {
	Version     string
	LogLevel    string
	LogFilename string
	Lanes       []LaneSettings
}

func LoadConfig(path string) (*AppSettings, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("json")

	// Defaults for distributions
	viper.SetDefault("Lanes.0.RPM.GammaDistribution", []float64{0.25, 0.25, 0.25, 0.25})
	viper.SetDefault("Lanes.0.RPM.NeutronDistribution", []float64{0.25, 0.25, 0.25, 0.25})

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file: %v. Creating default.", err)
		// Provide default
		settings := &AppSettings{
			Version:     "1.0.0",
			LogLevel:    "INFO",
			LogFilename: "rpm_simulator.log",
			Lanes: []LaneSettings{
				{
					LaneID: 1, LaneName: "Default Lane", Enabled: true,
					AutoGammaProbability: 0.1, AutoNeutronProbability: 0.05,
					AutoInterval: 30,
					RPM: RPMSettings{
						IPAddr: "127.0.0.1", Port: 10001,
						GammaBG: 250, NeutronBG: 2, GammaNSigma: 6,
						NeutronThreshold: 5, GHThreshold: 450,
						GLThreshold: 80, NHThreshold: 10,
						GammaDistribution: []float64{0.25, 0.25, 0.25, 0.25},
						NeutronDistribution: []float64{0.25, 0.25, 0.25, 0.25},
					},
				},
			},
		}
		return settings, nil
	}

	var settings AppSettings
	if err := viper.Unmarshal(&settings); err != nil {
		return nil, err
	}

	// Make sure distributions aren't empty
	for i := range settings.Lanes {
		if len(settings.Lanes[i].RPM.GammaDistribution) == 0 {
			settings.Lanes[i].RPM.GammaDistribution = []float64{0.25, 0.25, 0.25, 0.25}
		}
		if len(settings.Lanes[i].RPM.NeutronDistribution) == 0 {
			settings.Lanes[i].RPM.NeutronDistribution = []float64{0.25, 0.25, 0.25, 0.25}
		}
	}

	return &settings, nil
}
