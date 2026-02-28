package singal

type NumStreams struct {
	OS  int `json:"OS"`
	MIS int `json:"MIS"`
}

type SCTPCapabilities struct {
	NumStreams NumStreams `json:"numStreams"`
}

type AppData struct {
	Direction string `json:"direction"`
	Channel   string `json:"channel"`
	Source    string `json:"source"`
	PeerId    string `json:"peerId"`
}

type CreateTransportReqData struct {
	SCTPCapabilities SCTPCapabilities `json:"sctpCapabilities"`
	ForceTCP         bool             `json:"forceTcp"`
	AppData          AppData          `json:"appData"`
}

type ICEParameters struct {
	UsernameFragment string `json:"usernameFragment"`
	Password         string `json:"password"`
	ICELite          bool   `json:"iceLite"`
}

type ICECandidate struct {
	Foundation string `json:"foundation"`
	Priority   int    `json:"priority"`
	IP         string `json:"ip"`
	Address    string `json:"address"`
	Protocol   string `json:"protocol"`
	Port       int    `json:"port"`
	Type       string `json:"type"`
	TCPType    string `json:"tcpType,omitempty"`
}

type DTLSParameters struct {
	Fingerprints []Fingerprint `json:"fingerprints"`
	Role         string        `json:"role"`
}

type Fingerprint struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type SCTPParameters struct {
	Port               int  `json:"port"`
	OS                 int  `json:"OS"`
	MIS                int  `json:"MIS"`
	MaxMessageSize     int  `json:"maxMessageSize"`
	SendBufferSize     int  `json:"sendBufferSize"`
	SCTPBufferedAmount int  `json:"sctpBufferedAmount"`
	IsDataChannel      bool `json:"isDataChannel"`
}

type CreateTransportResData struct {
	TransportID    string         `json:"transportId"`
	ICEParameters  ICEParameters  `json:"iceParameters"`
	ICECandidates  []ICECandidate `json:"iceCandidates"`
	DTLSParameters DTLSParameters `json:"dtlsParameters"`
	SCTPParameters SCTPParameters `json:"sctpParameters"`
}

type Device struct {
	Flag    string `json:"flag"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RtpCapabilities struct {
	Mid         string              `json:"mid"`
	MediaCodecs []CodecCapabilities `json:"codecs"`
}

type JoinReqData struct {
	DisplayName     string          `json:"displayName"`
	Device          Device          `json:"device"`
	RtpCapabilities RtpCapabilities `json:"rtpCapabilities"`
}

type JoinResData struct {
	Peers []*PeerData `json:"peers"`
}

type ConnectTransportReqData struct {
	TransportId    string         `json:"transportId"`
	DTLSParameters DTLSParameters `json:"dtlsParameters"`
}

type SCTPStreamParameters struct {
	StreamId int  `json:"streamId"`
	Orderd   bool `json:"orderd"`
}

type ProduceDataReqData struct {
	TransportId          string               `json:"transportId"`
	SCTPStreamParameters SCTPStreamParameters `json:"sctpStreamParameters"`
	Label                string               `json:"label"`
	AppData              AppData              `json:"appData"`
}

type ProduceDataResData struct {
	DataProducerId string `json:"dataProducerId,omitempty"`
}

type NewDataConsumerReqData struct {
	PeerId               string               `json:"peerId"`
	TransportId          string               `json:"transportId"`
	DataProducerId       string               `json:"dataProducerId"`
	DataConsumerId       string               `json:"dataConsumerId"`
	SCTPStreamParameters SCTPStreamParameters `json:"sctpStreamParameters"`
	Label                string               `json:"label"`
	AppData              AppData              `json:"appData"`
}

type NewConsumerReqData struct {
	PeerId           string            `json:"peerId"`
	TransportId      string            `json:"transportId"`
	ConsumerId       string            `json:"consumerId"`
	ProducerId       string            `json:"producerId"`
	Kind             string            `json:"kind"`
	RtpParameters    RtpCapabilities   `json:"rtpParameters"`
	HeaderExtensions []HeaderExtension `json:"headerExtensions"`
	Encodings        []Encodings       `json:"encodings"`
}

type Rtcp struct {
	CName       string `json:"cname"`
	ReducedSize bool   `json:"reducedSize"`
}

type Encodings struct {
	Active                bool   `json:"active,omitempty"`
	ScalabilityMode       string `json:"scalabilityMode,omitempty"`
	ScaleResolutionDownBy int    `json:"scaleResolutionDownBy,omitempty"`
	MaxBitrate            int    `json:"maxBitrate,omitempty"`
	Rid                   string `json:"rid,omitempty"`
	Dtx                   bool   `json:"dtx,omitempty"`
	Ssrc                  uint32 `json:"ssrc,omitempty"`
}

type ProduceReqData struct {
	TransportId      string            `json:"transportId"`
	Kind             string            `json:"kind"`
	RtpParameters    RtpCapabilities   `json:"rtpParameters"`
	HeaderExtensions []HeaderExtension `json:"headerExtensions"`
	Encodings        []Encodings       `json:"encodings"`
	Rtcp             Rtcp              `json:"rtcp"`
	Msid             string            `json:"msid"`
}

type ProduceResData struct {
	ProducerId string `json:"producerId"`
}

type PeerData struct {
	PeerId        string `json:"peerId"`
	DisplayName   string `json:"displayName"`
	Device        Device `json:"device"`
	RemoteAddress string `json:"remoteAddress"`
}
