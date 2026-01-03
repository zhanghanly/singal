package singal

import (
	"github.com/gorilla/websocket"
	"time"
)

type User struct {
	userId           string
	createTs         int64
	wsConn           *websocket.Conn
	roomId           string
	node             *SfuNode
	videoProducerId  string
	audioProducerId  string
	videoConsumerIds []string
	audioConsumerIds []string
}

func NewUser(conn *websocket.Conn) *User {
	return &User{
		wsConn:           conn,
		createTs:         time.Now().Unix(),
		videoConsumerIds: make([]string, 0),
		audioConsumerIds: make([]string, 0),
	}
}
