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

func (r *Room) GetOtherUsers(user *User) []*User {
	userLst := make([]*User, 0)
	for k, v := range r.users {
		if k == user.userId {
			continue
		}

		userLst = append(userLst, v)
	}

	return userLst
}

func (r *Room) CreateWebrtcTransport(req *CreateTransportReqData) (*CreateTransportResData, error) {
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
	}

	for _, v := range res.IceCandidates {
		candidate := ICECandidate{
			Foundation: v.Foundation,
			Priority:   int(v.Priority),
			IP:         v.Ip,
			Protocol:   v.Protocol,
			Port:       int(v.Port),
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
		producer := &Producer{
			id:          RandString(12),
			transportId: res.TransportId,
		}
		r.router.addProducer(producer)

	case "consumer":
		consumer := &Consumer{
			id:          RandString(12),
			transportId: res.TransportId,
		}
		r.router.addConsumer(consumer)

	default:
		return nil, errors.New("bad AppData direction")
	}

	return transportResData, nil
}

func (r *Room) ConnectWebrtcTransport(req *ConnectTransportReqData) error {
	return gRtcServer.ConnectWebrtcTransport(r.router, req)
}
