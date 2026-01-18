package singal

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type WsRequest struct {
	Request bool        `json:"request"`
	Id      int         `json:"id"`
	Method  string      `json:"method"`
	Data    interface{} `json:"data"`
}

type WsResponse struct {
	Response bool        `json:"response"`
	Id       int         `json:"id"`
	Ok       bool        `json:"method"`
	Data     interface{} `json:"data"`
}

type WsNotification struct {
	Notification bool        `json:"notification"`
	Method       string      `json:"method"`
	Data         interface{} `json:"data"`
}

type User struct {
	userId           string
	peerId           string
	displayName      string
	createTs         int64
	wsConn           *websocket.Conn
	wsServer         *WsServer
	sendMsg          chan []byte
	roomId           string
	node             *SfuNode
	videoProducerId  string
	audioProducerId  string
	videoConsumerIds []string
	audioConsumerIds []string
}

func NewUser(conn *websocket.Conn, server *WsServer, peerid string, roomid string) *User {
	return &User{
		wsConn:           conn,
		wsServer:         server,
		peerId:           peerid,
		roomId:           roomid,
		createTs:         time.Now().Unix(),
		sendMsg:          make(chan []byte),
		videoConsumerIds: make([]string, 0),
		audioConsumerIds: make([]string, 0),
	}
}

func (u *User) ReadMessage() {
	defer func() {
		u.wsServer.Unregister <- u
		u.wsConn.Close()
		logger.Infof("close connection from read")
	}()

	for {
		_, messageBytes, err := u.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Infof("read message failed: %v", err)
			}
			break
		}
		logger.Infof("read msg=%s", string(messageBytes))

		var wsReq WsRequest
		if err := json.Unmarshal(messageBytes, &wsReq); err != nil {
			logger.Infof("JSON decode failed: %v", err)
			continue
		}

		switch wsReq.Method {
		case "getRouterRtpCapabilities":
			logger.Infof("recv getRouterRtpCapabilities message")
			//u.wsServer.Broadcast <- msg

		case "createWebRtcTransport":
			logger.Infof("recv createWebRtcTransport message")
			//u.sendMsg <- response

		case "connectWebRtcTransport":
			logger.Infof("recv connectWebRtcTransport message")
			//u.wsServer.Broadcast <- msg

		case "join":
			logger.Infof("recv join message")
			//u.wsServer.Broadcast <- msg

		case "produceData":
			logger.Infof("recv produceData message")
			//u.wsServer.Broadcast <- msg

		case "produce":
			logger.Infof("recv produce message")
			//u.wsServer.Broadcast <- msg

		default:
			logger.Infof("unknown message type:")
		}
	}
}

func (u *User) WriteMessage() {
	defer func() {
		u.wsConn.Close()
		logger.Infof("close connection from write")
	}()

	for message := range u.sendMsg {
		if err := u.wsConn.WriteMessage(websocket.TextMessage, message); err != nil {
			logger.Infof("write message failed: %v", err)
			return
		}
	}
}
