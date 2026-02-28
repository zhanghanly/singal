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

func (r *Room) GetOtherUsers(user *User) []*PeerData {
	userLst := make([]*PeerData, 0)
	for k, v := range r.users {
		if k == user.userId {
			continue
		}

		peerData := &PeerData{
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
			PeerId:        user.PeerId,
			DisplayName:   user.DisplayName,
			Device:        user.Device,
			RemoteAddress: user.RemoteAddress,
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

func (r *Room) ReqOtherNewConsumer(userId string) {
	for k, v := range r.users {
		if k == userId {
			continue
		}

		producers := r.router.getProducerById(userId)
		for _, producer := range producers {
			consumers := r.router.getConsumerById(userId)
			for _, consumer := range consumers {
				if consumer.kind != producer.kind {
					continue
				}
				if consumer != nil {
					newConsumerData := &NewConsumerReqData{
						PeerId:           v.PeerId,
						TransportId:      consumer.transportId,
						ConsumerId:       consumer.consumerId,
						ProducerId:       producer.producerId,
						Kind:             consumer.kind,
						RtpParameters:    producer.parameters.RtpParameters,
						HeaderExtensions: producer.parameters.HeaderExtensions,
						Encodings:        producer.parameters.Encodings,
					}
					v.RequestNewConsumer(newConsumerData)
				}
			}
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

func (r *Room) Produce(userId string, req *ProduceReqData) (string, error) {
	producer := &Producer{
		producerId:  RandString(36),
		transportId: req.TransportId,
		kind:        req.Kind,
		parameters:  req,
	}
	r.router.addProducer(userId, producer)

	if len(r.router.getProducerById(userId)) > 1 {
		r.ReqOtherNewConsumer(userId)
	}

	return producer.producerId, nil
}
