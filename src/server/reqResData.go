package singal

type NumStreams struct {
	OS  int `json:"OS"`
	MIS int `json:"MIS"`
}

type SCTPCapabilities struct {
	NumStreams NumStreams `json:"numStreams"`
}

type AppData struct {
	Direction string `json:"direction,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Source    string `json:"source,omitempty"`
	PeerId    string `json:"peerId,omitempty"`
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
	Mid              string                  `json:"mid"`
	MediaCodecs      []CodecCapabilities     `json:"codecs"`
	HeaderExtensions []ProducerHeadExtension `json:"headerExtensions"`
	Encodings        []*Encodings            `json:"encodings"`
	Rtcp             Rtcp                    `json:"rtcp"`
	Msid             string                  `json:"msid"`
}

type JoinReqData struct {
	DisplayName     string          `json:"displayName"`
	Device          *Device         `json:"device"`
	RtpCapabilities RtpCapabilities `json:"rtpCapabilities"`
}

type JoinResData struct {
	Peers []*Peer `json:"peers"`
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

type ConsumerScore struct {
	Score          int   `json:"score"`
	ProducerScore  int   `json:"producerScore"`
	ProducerScores []int `json:"producerScores"`
}

type NewConsumerReqData struct {
	PeerId         string          `json:"peerId"`
	TransportId    string          `json:"transportId"`
	ConsumerId     string          `json:"consumerId"`
	ProducerId     string          `json:"producerId"`
	Kind           string          `json:"kind"`
	RtpParameters  RtpCapabilities `json:"rtpParameters"`
	Type           string          `json:"type"`
	ProducerPaused bool            `json:"producerPaused"`
	AppData        AppData         `json:"appData"`
	ConsumerScore  *ConsumerScore  `json:"consumerScore"`
}

type Rtcp struct {
	CName       string `json:"cname"`
	ReducedSize bool   `json:"reducedSize"`
}

type Rtx struct {
	Ssrc uint32 `json:"ssrc"`
}

type Encodings struct {
	Active                bool   `json:"active,omitempty"`
	ScalabilityMode       string `json:"scalabilityMode,omitempty"`
	ScaleResolutionDownBy int    `json:"scaleResolutionDownBy,omitempty"`
	MaxBitrate            int    `json:"maxBitrate,omitempty"`
	Rid                   string `json:"rid,omitempty"`
	Dtx                   bool   `json:"dtx,omitempty"`
	Ssrc                  uint32 `json:"ssrc,omitempty"`
	Rtx                   *Rtx   `json:"rtx,omitempty"`
}

type ProducerHeadExtension struct {
	Uri        string      `json:"uri"`
	Id         int         `json:"id"`
	Encrypt    bool        `json:"encrypt"`
	Parameters interface{} `json:"parameters"`
}

type ProduceReqData struct {
	TransportId   string          `json:"transportId"`
	Kind          string          `json:"kind"`
	RtpParameters RtpCapabilities `json:"rtpParameters"`
}

type ProduceResData struct {
	ProducerId string `json:"producerId"`
}

type Peer struct {
	PeerId        string  `json:"peerId,omitempty"`
	DisplayName   string  `json:"displayName,omitempty"`
	Device        *Device `json:"device,omitempty"`
	RemoteAddress string  `json:"remoteAddress,omitempty"`
}

type PeerData struct {
	Peer Peer `json:"peer"`
}
