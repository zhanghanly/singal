package singal

import (
	"encoding/json"
	"os"
)

type CodecCapabilities struct {
	Kind                 string           `json:"kind,omitempty"`
	MimeType             string           `json:"mimeType"`
	ClockRate            int              `json:"clockRate"`
	Channels             int              `json:"channels,omitempty"`
	Parameters           *CodecParameters `json:"parameters,omitempty"`
	Feedbacks            []RtcpFeedback   `json:"rtcpFeedback"`
	PreferredPayloadType int              `json:"preferredPayloadType,omitempty"`
	PayloadType          int              `json:"payloadType,omitempty"`
}

type CodecParameters struct {
	PacketizationMode     int    `json:"packetization-mode,omitempty"`
	ProfileLevelId        string `json:"profile-level-id,omitempty"`
	LevelAsymmetryAllowed int    `json:"level-asymmetry-allowed,omitempty"`
	XGoogleStartBitrate   int    `json:"x-google-start-bitrate,omitempty"`
	Apt                   int    `json:"apt,omitempty"`
}

type RtcpFeedback struct {
	Type      string `json:"type"`
	Parameter string `json:"parameter"`
}

type HeaderExtension struct {
	Kind             string `json:"kind"`
	Uri              string `json:"uri"`
	PreferredId      int    `json:"preferredId"`
	PreferredEncrypt bool   `json:"preferredEncrypt"`
	Direction        string `json:"direction"`
}

type Https struct {
	Addr string `json:"addr"`
	Cert string `json:"cert,omitempty"`
	Key  string `json:"key,omitempty"`
}

type RouterRtpCapabilities struct {
	MediaCodecs      []CodecCapabilities `json:"codecs"`
	HeaderExtensions []HeaderExtension   `json:"headerExtensions"`
}

type Config struct {
	RtpCapabilities RouterRtpCapabilities `json:"routerRtpCapabilities"`
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

	//logger.Info(len(gConfig.MediaCodecs))
	//jsonData, err := json.Marshal(gConfig)
	//if err != nil {
	//	logger.Info("failed: ")
	//	return err
	//}
	//os.WriteFile("test.json", jsonData, 0644)

	return nil
}
