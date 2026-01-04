package singal

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type User struct {
	userId           string
	createTs         int64
	wsConn           *websocket.Conn
	wsServer         *WsServer
	sendMsg          chan Message
	roomId           string
	node             *SfuNode
	videoProducerId  string
	audioProducerId  string
	videoConsumerIds []string
	audioConsumerIds []string
}

func NewUser(conn *websocket.Conn, server *WsServer) *User {
	return &User{
		wsConn:           conn,
		wsServer:         server,
		createTs:         time.Now().Unix(),
		sendMsg:          make(chan Message, 256),
		videoConsumerIds: make([]string, 0),
		audioConsumerIds: make([]string, 0),
	}
}

func (u *User) ReadMessage() {
	defer func() {
		u.wsServer.Unregister <- u
		u.wsConn.Close()
	}()

	for {
		_, messageBytes, err := u.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Infof("read message failed: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			logger.Infof("JSON decode failed: %v", err)
			continue
		}

		msg.Sender = u.userId
		msg.Time = time.Now().Unix()

		switch msg.Type {
		case "chat":
			logger.Infof("recv chat message: %s", msg.Content)
			u.wsServer.Broadcast <- msg

		case "ping":
			response := Message{
				Type:    "pong",
				Content: "pong",
				Sender:  "server",
				Time:    time.Now().Unix(),
			}
			u.sendMsg <- response

		case "getOrCreateRoom":
			u.wsServer.Broadcast <- msg

		default:
			logger.Infof("unknown message type: %s", msg.Type)
		}
	}
}

func (u *User) WriteMessage() {
	defer func() {
		u.wsConn.Close()
	}()

	for {
		select {
		case message, ok := <-u.sendMsg:
			if !ok {
				u.wsConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			jsonMsg, err := json.Marshal(message)
			if err != nil {
				logger.Infof("JSON encode failed: %v", err)
				continue
			}

			if err := u.wsConn.WriteMessage(websocket.TextMessage, jsonMsg); err != nil {
				logger.Infof("write message failed: %v", err)
				return
			}

		default:
			logger.Infof("write message failed:")
		}
	}
}
