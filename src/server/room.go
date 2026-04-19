package singal

import (
	"errors"
	"strings"
	"time"
)

const (
	MID int = iota + 1
	RTP_STREAM_ID
	REPAIRED_RTP_STREAM_ID
	ABS_SEND_TIME
	TRANSPORT_WIDE_CC_01
	SSRC_AUDIO_LEVEL
	DEPENDENCY_DESCRIPTOR
	VIDEO_ORIENTATION
	TIME_OFFSET
	ABS_CAPTURE_TIME
	PLAYOUT_DELAY
	MEDIASOUP_PACKET_ID
)

func ChangeExtensionId(extensionIds []ProducerHeadExtension) {
	for k, _ := range extensionIds {
		extensionIds[k].Id = GetNewExtensionId(extensionIds[k].Uri)
	}
}

func GetNewExtensionId(uri string) int {
	if strings.Contains(uri, "mid") {
		return MID

	} else if strings.Contains(uri, "abs-send-time") {
		return ABS_SEND_TIME

	} else if strings.Contains(uri, "transport-wide-cc-extensions") {
		return TRANSPORT_WIDE_CC_01

	} else if strings.Contains(uri, "ssrc-audio-level") {
		return SSRC_AUDIO_LEVEL

	} else if strings.Contains(uri, "abs-capture-time") {
		return ABS_CAPTURE_TIME

	} else if strings.Contains(uri, "video-orientation") {
		return VIDEO_ORIENTATION

	} else if strings.Contains(uri, "toffset") {
		return TIME_OFFSET

	} else if strings.Contains(uri, "playout-delay") {
		return PLAYOUT_DELAY

	} else if strings.Contains(uri, "repaired-rtp-stream-id") {
		return REPAIRED_RTP_STREAM_ID

	} else if strings.Contains(uri, "rtp-stream-id") {
		return RTP_STREAM_ID
	}

	return -1
}

func RemoveFeedbackRemb(feedback []RtcpFeedback) {
	for i := 0; i < len(feedback); i++ {
		if feedback[i].Type == "goog-remb" {
			feedback = append(feedback[:i], feedback[i+1:]...)
			i--
		}
	}
}

type Room struct {
	roomId   string
	createTs int64
	router   *Router
	users    map[string]*User
}

func NewRoom(id string) *Room {
	route, err := gRtcServer.CreateRouterOnWorker()
	if err != nil {
		logger.Errorf("create router failed, reason=%v", err)
		return nil
	}
	return &Room{
		roomId:   id,
		createTs: time.Now().Unix(),
		router:   route,
		users:    make(map[string]*User),
	}
}

func (r *Room) AddUser(user *User) {
	if _, ok := r.users[user.userId]; !ok {
		r.users[user.userId] = user

		logger.Infof("add userId=%s peerId=%s to roomId=%s", user.userId, user.PeerId, r.roomId)
	}
}

func (r *Room) GetUserNums() int {
	return len(r.users)
}

func (r *Room) DeleteUser(user *User) {
	delete(r.users, user.userId)
	logger.Infof("delete userId=%s peerId=%s from roomId=%s", user.userId, user.PeerId, r.roomId)

	//notify other users
	peer := &Peer{
		PeerId: user.PeerId,
	}
	for _, user := range r.users {
		user.NotifyPeerClosed(peer)
	}
}

func (r *Room) GetOtherUsers(user *User) []*Peer {
	userLst := make([]*Peer, 0)
	for k, v := range r.users {
		if k == user.userId {
			continue
		}

		peerData := &Peer{
			PeerId:        v.PeerId,
			DisplayName:   v.DisplayName,
			Device:        v.Device,
			RemoteAddress: v.RemoteAddress,
		}
		userLst = append(userLst, peerData)
	}

	return userLst
}

func (r *Room) NotifyOtherUsers(user *User) {
	for k, v := range r.users {
		if k == user.userId {
			continue
		}

		peerData := &PeerData{
			Peer: Peer{
				PeerId:        user.PeerId,
				DisplayName:   user.DisplayName,
				Device:        user.Device,
				RemoteAddress: user.RemoteAddress,
			},
		}
		v.NotifyNewPeer(peerData)
	}
}

func (r *Room) ReqOtherNewDataConsumer(user *User) {
	for k, v := range r.users {
		if k == user.userId {
			continue
		}

		newDataConsumerData := &NewDataConsumerReqData{
			PeerId:         user.PeerId,
			TransportId:    r.router.getConsumerTransportId(user.userId),
			DataProducerId: RandString(36),
			DataConsumerId: RandString(36),
			SCTPStreamParameters: SCTPStreamParameters{
				StreamId: 0,
				Orderd:   true,
			},
			Label: "chat",
			AppData: AppData{
				PeerId:  user.PeerId,
				Channel: "chat",
			},
		}

		v.RequestNewDataConsumer(newDataConsumerData)
	}
}

func (r *Room) CreateNewConsumerData(consumer *Consumer, producer *Producer, peerId string, userId string) *NewConsumerReqData {
	newConsumerData := &NewConsumerReqData{
		PeerId:         peerId,
		TransportId:    consumer.transportId,
		ConsumerId:     consumer.consumerId,
		ProducerId:     consumer.producerId,
		Kind:           consumer.kind,
		RtpParameters:  producer.parameters.RtpParameters,
		ProducerPaused: false,
		Type:           "simple",
		AppData: AppData{
			PeerId: peerId,
			Source: "audio",
		},
		ConsumerScore: &ConsumerScore{
			Score:          10,
			ProducerScore:  0,
			ProducerScores: []int{0},
		},
	}
	//reset Mid
	newConsumerData.RtpParameters.Mid = r.router.getConsumerStreamMid(userId)

	if consumer.kind == "video" {
		newConsumerData.RtpParameters.Rtcp.CName = RandString(20)

		newConsumerData.RtpParameters.MediaCodecs[0].PayloadType = 101
		newConsumerData.RtpParameters.MediaCodecs[1].PayloadType = 102
		newConsumerData.RtpParameters.MediaCodecs[1].Parameters.Apt = 101

		//newConsumerData.Type = "simulcast"
		newConsumerData.AppData.Source = "video"

		newConsumerData.ConsumerScore.ProducerScores = append(newConsumerData.ConsumerScore.ProducerScores, 0)
		newConsumerData.ConsumerScore.ProducerScores = append(newConsumerData.ConsumerScore.ProducerScores, 0)

	} else {
		newConsumerData.RtpParameters.MediaCodecs[0].PayloadType = 100
	}

	_, err := gRtcServer.CreateConsumer(r.router, newConsumerData)
	if err != nil {
		logger.Errorf("create consumer failed, reason=%v", err)
	}

	ChangeExtensionId(newConsumerData.RtpParameters.HeaderExtensions)

	return newConsumerData
}

func (r *Room) ReqPullOtherNewProducer(u *User, producer *Producer) {
	for k, v := range r.users {
		if k == u.userId {
			continue
		}

		producers := r.router.getProducerById(k)
		for _, producer1 := range producers {
			if producer1.kind != producer.kind {
				continue
			}

			consumer := &Consumer{
				consumerId:  RandString(36),
				transportId: r.router.getConsumerTransportId(u.userId),
				kind:        producer.kind,
				producerId:  producer1.producerId,
			}

			success := r.router.addConsumer(u.userId, consumer)
			if success {
				newConsumerData := r.CreateNewConsumerData(consumer, producer1, v.PeerId, u.userId)
				u.RequestNewConsumer(newConsumerData)
			}
		}
	}
}

func (r *Room) ReqOtherNewConsumer(u *User, producer *Producer) {
	for k, v := range r.users {
		if k == u.userId {
			continue
		}

		consumer := &Consumer{
			consumerId:  RandString(36),
			transportId: r.router.getConsumerTransportId(v.userId),
			kind:        producer.kind,
			producerId:  producer.producerId,
		}

		newConsumerData := r.CreateNewConsumerData(consumer, producer, u.PeerId, v.userId)
		v.RequestNewConsumer(newConsumerData)
		r.router.addConsumer(v.userId, consumer)
	}
}

func (r *Room) CreateWebrtcTransport(req *CreateTransportReqData, u *User) (*CreateTransportResData, error) {
	res, err := gRtcServer.CreateWebrtcTransport(r.router)
	if err != nil {
		logger.Errorf("grpc CreateWebrtcTransport failed, reason=%v", err)
		return nil, errors.New("not res from peer")
	}

	transportResData := &CreateTransportResData{
		TransportID: res.TransportId,
		ICEParameters: ICEParameters{
			UsernameFragment: res.IceUfrag,
			Password:         res.IcePwd,
			ICELite:          true,
		},
		ICECandidates: make([]ICECandidate, 0),
		DTLSParameters: DTLSParameters{
			Role:         "auto",
			Fingerprints: make([]Fingerprint, 0),
		},
		SCTPParameters: SCTPParameters{
			Port:               5000,
			OS:                 1024,
			MIS:                1024,
			MaxMessageSize:     262144,
			SendBufferSize:     262144,
			SCTPBufferedAmount: 0,
			IsDataChannel:      true,
		},
	}

	for _, v := range res.IceCandidates {
		candidate := ICECandidate{
			Foundation: v.Foundation,
			Priority:   int(v.Priority),
			IP:         v.Ip,
			Address:    v.Ip,
			Protocol:   v.Protocol,
			Port:       int(v.Port),
			Type:       "host",
		}

		transportResData.ICECandidates = append(transportResData.ICECandidates, candidate)
	}
	for _, v := range res.DtlsFingerprints {
		fingerprint := Fingerprint{
			Algorithm: v.Algorithm,
			Value:     v.Value,
		}

		transportResData.DTLSParameters.Fingerprints = append(transportResData.DTLSParameters.Fingerprints, fingerprint)
	}

	switch req.AppData.Direction {
	case "producer":
		r.router.saveProducerTransportId(u.userId, res.TransportId)

	case "consumer":
		r.router.saveConsumeTransportId(u.userId, res.TransportId)

	default:
		return nil, errors.New("bad AppData direction")
	}

	return transportResData, nil
}

func (r *Room) CreateNewDataConsumer(u *User, transportId string) *NewDataConsumerReqData {
	if r.router.getConsumerTransportId(u.userId) == transportId {
		return nil
	}

	return &NewDataConsumerReqData{
		TransportId:    r.router.getConsumerTransportId(u.userId),
		DataProducerId: RandString(36),
		DataConsumerId: RandString(36),
		SCTPStreamParameters: SCTPStreamParameters{
			StreamId: 0,
			Orderd:   true,
		},
		Label: "bot",
		AppData: AppData{
			Channel: "bot",
		},
	}
}

func (r *Room) ConnectWebrtcTransport(req *ConnectTransportReqData) error {
	return gRtcServer.ConnectWebrtcTransport(r.router, req)
}

func (r *Room) Produce(u *User, req *ProduceReqData) (string, error) {
	producer := &Producer{
		producerId:  RandString(36),
		transportId: req.TransportId,
		kind:        req.Kind,
		parameters:  req,
	}
	r.router.addProducer(u.userId, producer)

	// remove remb feedback
	for _, codec := range producer.parameters.RtpParameters.MediaCodecs {
		RemoveFeedbackRemb(codec.Feedbacks)
	}

	_, err := gRtcServer.CreateProducer(r.router, producer)
	if err != nil {
		logger.Errorf("create producer failed, reason=%v", err)
	}

	r.ReqOtherNewConsumer(u, producer)
	r.ReqPullOtherNewProducer(u, producer)

	return producer.producerId, nil
}

func (r *Room) CloseProducer(u *User, producerId string) error {
	producer := r.router.GetProducer(u.userId, producerId)
	if producer == nil {
		return errors.New("producer not found")
	}

	_, err := gRtcServer.CloseProducer(r.router, producer)
	if err != nil {
		logger.Errorf("close producer failed, reason=%v", err)
	}
	r.router.removeProducer(u.userId, producerId)

	//consumers := r.router.GetConsumerByProducerId(u.userId, producerId)
	//for _, consumer := range consumers {
	//	_, err := gRtcServer.CloseConsumer(r.router, consumer)
	//	if err != nil {
	//		logger.Errorf("close consumer failed, reason=%v", err)
	//	}
	//}
	closedConsumers := r.router.removeConsumerByProducerId(u.userId, producerId)
	for userId, consumerId := range closedConsumers {
		if _, exist := r.users[userId]; exist {
			data := &ConsumeResData{
				ConsumerId: consumerId,
			}

			r.users[userId].NotifyConsumerClosed(data)
		}
	}

	return nil
}

func (r *Room) Close() {
	err := gRtcServer.CloseRouterOnWorker(r.router)
	if err != nil {
		logger.Errorf("close router failed, reason=%v", err)
	}
}
