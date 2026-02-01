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

		logger.Infof("add userId=%s peerId=%s to roomId=%s", user.userId, user.peerId, r.roomId)
	}
}

func (r *Room) DeleteUser(user *User) {
	delete(r.users, user.userId)
	logger.Infof("delete userId=%s peerId=%s from roomId=%s", user.userId, user.peerId, r.roomId)
}

func (r *Room) CreateWebrtcTransport(req *CreateTransportReqData) (*CreateTransportResData, error) {
	res, err := gRtcServer.CreateWebrtcTransport(r.router)
	if err != nil {
		return nil, errors.New("not res from peer")
	}

	transportResData := &CreateTransportResData{
		TransportID: res.TransportId,
		ICEParameters: ICEParameters{
			UsernameFragment: res.IceUfrag,
			Password:         res.IcePwd,
			ICELite:          true,
		},
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

func (r *Room) ConnectWebrtcTransport(user *User) {

}
