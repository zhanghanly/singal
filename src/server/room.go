package singal

import (
	"errors"
	"time"
)

type Room struct {
	roomId   string
	createTs int64
	router   *Router
	users    map[string]*User
}

func NewRoom(id string, route *Router) *Room {
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

func (r *Room) DeleteUser(user *User) {
	delete(r.users, user.userId)
	logger.Infof("delete userId=%s peerId=%s from roomId=%s", user.userId, user.PeerId, r.roomId)
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

func (r *Room) ReqPullOtherNewProducer(u *User, producer *Producer) {
	var consumer *Consumer
	consumers := r.router.getConsumerById(u.userId)
	for _, v := range consumers {
		if v.kind == producer.kind {
			consumer = v
		}
	}

	for k, v := range r.users {
		if k == u.userId {
			continue
		}

		producers := r.router.getOtherProducersById(u.userId)
		for _, producer1 := range producers {
			if producer1.kind != producer.kind {
				continue
			}
			newConsumerData := &NewConsumerReqData{
				PeerId:         v.PeerId,
				TransportId:    consumer.transportId,
				ConsumerId:     consumer.consumerId,
				ProducerId:     producer1.producerId,
				Kind:           consumer.kind,
				RtpParameters:  producer1.parameters.RtpParameters,
				ProducerPaused: false,
				Type:           "simple",
				AppData: AppData{
					PeerId: u.PeerId,
					Source: "audio",
				},
				ConsumerScore: &ConsumerScore{
					Score:          10,
					ProducerScore:  0,
					ProducerScores: []int{0},
				},
			}
			if consumer.kind == "video" {
				newConsumerData.RtpParameters.Rtcp.CName = RandString(20)

				newConsumerData.RtpParameters.MediaCodecs[0].PayloadType = 101
				newConsumerData.RtpParameters.MediaCodecs[1].PayloadType = 102
				newConsumerData.RtpParameters.MediaCodecs[1].Parameters.Apt = 101

				newConsumerData.Type = "simulcast"
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

			if len(newConsumerData.RtpParameters.Encodings) > 1 {
				newConsumerData.RtpParameters.Encodings = newConsumerData.RtpParameters.Encodings[:1]
				newConsumerData.RtpParameters.Encodings[0].ScalabilityMode = "L3T3"
				newConsumerData.RtpParameters.Encodings[0].Rid = ""
				newConsumerData.RtpParameters.Encodings[0].ScaleResolutionDownBy = 0
				newConsumerData.RtpParameters.Encodings[0].Active = false
			}

			u.RequestNewConsumer(newConsumerData)
		}
	}
}

func (r *Room) ReqOtherNewConsumer(u *User, producer *Producer) {
	for k, v := range r.users {
		if k == u.userId {
			continue
		}

		consumers := r.router.getOtherConsumersById(u.userId)
		for _, consumer := range consumers {
			if consumer.kind != producer.kind {
				continue
			}
			newConsumerData := &NewConsumerReqData{
				PeerId:        u.PeerId,
				TransportId:   consumer.transportId,
				ConsumerId:    consumer.consumerId,
				ProducerId:    producer.producerId,
				Kind:          consumer.kind,
				RtpParameters: producer.parameters.RtpParameters,
				//Msid:          producer.parameters.Msid,
				//Rtcp:          producer.parameters.Rtcp,
				//HeaderExtensions: producer.parameters.RtpParameters.HeaderExtensions,
				//Encodings: producer.parameters.Encodings,
				ProducerPaused: false,
				Type:           "simple",
				AppData: AppData{
					PeerId: u.PeerId,
					Source: "audio",
				},
				ConsumerScore: &ConsumerScore{
					Score:          10,
					ProducerScore:  0,
					ProducerScores: []int{0},
				},
			}
			if consumer.kind == "video" {
				//ssrc := rand.Uint32()
				//newConsumerData.RtpParameters.Encodings[0].Ssrc = ssrc
				//newConsumerData.RtpParameters.Encodings[0].Rtx = &Rtx{
				//	Ssrc: ssrc + 1,
				//}
				newConsumerData.RtpParameters.Rtcp.CName = RandString(20)

				newConsumerData.RtpParameters.MediaCodecs[0].PayloadType = 101
				newConsumerData.RtpParameters.MediaCodecs[1].PayloadType = 102
				newConsumerData.RtpParameters.MediaCodecs[1].Parameters.Apt = 101

				newConsumerData.Type = "simulcast"
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

			if len(newConsumerData.RtpParameters.Encodings) > 1 {
				newConsumerData.RtpParameters.Encodings = newConsumerData.RtpParameters.Encodings[:1]
				newConsumerData.RtpParameters.Encodings[0].ScalabilityMode = "L3T3"
				newConsumerData.RtpParameters.Encodings[0].Rid = ""
				newConsumerData.RtpParameters.Encodings[0].ScaleResolutionDownBy = 0
				newConsumerData.RtpParameters.Encodings[0].Active = false
			}

			v.RequestNewConsumer(newConsumerData)
		}
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
		r.router.CreateConsumes(u.userId)

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

	logger.Infof("producer.parameters.Encodings size=%d", len(producer.parameters.RtpParameters.Encodings))
	logger.Infof("producer.parameters.HeaderExtensions size=%d", len(producer.parameters.RtpParameters.HeaderExtensions))
	logger.Infof("producer.parameters.RtpParameters.MediaCodecs size=%d", len(producer.parameters.RtpParameters.MediaCodecs))

	if producer.kind == "audio" {
		_, err := gRtcServer.CreateProducer(r.router, producer)
		if err != nil {
			logger.Errorf("create producer failed, reason=%v", err)
		}

		r.ReqOtherNewConsumer(u, producer)
		r.ReqPullOtherNewProducer(u, producer)
	}

	return producer.producerId, nil
}
