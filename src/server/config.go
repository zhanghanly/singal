package singal

import (
	"encoding/json"
	"os"
)

type CodecCapabilities struct {
	Kind       string           `json:"kind"`
	MimeType   string           `json:"mimeType"`
	ClockRate  int              `json:"clockRate"`
	Channels   int              `json:"channels,omitempty"`
	Parameters *CodecParameters `json:"parameters,omitempty"`
}

type CodecParameters struct {
	PacketizationMode     int    `json:"packetization-mode,omitempty"`
	ProfileLevelId        string `json:"profile-level-id,omitempty"`
	LevelAsymmetryAllowed int    `json:"level-asymmetry-allowed,omitempty"`
	XGoogleStartBitrate   int    `json:"x-google-start-bitrate,omitempty"`
}

type Https struct {
	Port int    `json:"port"`
	Cert string `json:"cert,omitempty"`
	Key  string `json:"key,omitempty"`
}

type Config struct {
	MediaCodecs []CodecCapabilities `json:"mediaCodecs"`
	Http        Https               `json:"https"`
}

var gConfig *Config

func InitSetting() error {
	fileData, err := os.ReadFile("./config.json")
	if err != nil {
		logger.Info("read config.json failed:")
		return err
	}

	gConfig = &Config{}
	err = json.Unmarshal(fileData, gConfig)
	if err != nil {
		logger.Info("fail to yaml unmarshal:")
		return err
	}

	logger.Info(len(gConfig.MediaCodecs))
	//jsonData, err := json.Marshal(gConfig)
	//if err != nil {
	//	logger.Info("failed: ")
	//	return
	//}
	//os.WriteFile("test.json", jsonData, 0644)
	return nil
}
