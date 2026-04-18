package singal

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type WsMessage struct {
	Request      bool        `json:"request,omitempty"`
	Response     bool        `json:"response,omitempty"`
	Notification bool        `json:"notification,omitempty"`
	Ok           bool        `json:"ok"`
	Id           int         `json:"id"`
	Method       string      `json:"method,omitempty"`
	Data         interface{} `json:"data"`
}

type User struct {
	userId             string
	PeerId             string
	DisplayName        string
	createTs           int64
	wsConn             *websocket.Conn
	wsServer           *WsServer
	sendMsg            chan *WsMessage
	roomId             string
	Device             *Device
	RemoteAddress      string
	reqId              int
	newDataConsumerReq bool
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

		var wsReq WsMessage
		if err := json.Unmarshal(messageBytes, &wsReq); err != nil {
			logger.Infof("JSON decode failed: %v", err)
			continue
		}

		switch wsReq.Method {
		case "getRouterRtpCapabilities":
			u.handleGetRouterRtpCapabilities(&wsReq)

		case "createWebRtcTransport":
			u.handleCreateWebrtcTransport(&wsReq)

		case "connectWebRtcTransport":
			u.handleConnectWebrtcTransport(&wsReq)

		case "join":
			u.handleJoin(&wsReq)

		case "produceData":
			u.handleProduceData(&wsReq)

		case "produce":
			u.handleProduce(&wsReq)

		case "closeProducer":
			u.HandleCloseProducer(&wsReq)

		default:
			logger.Infof("unknown message type:=%s", wsReq.Method)
		}
	}
}

func (u *User) handleGetRouterRtpCapabilities(req *WsMessage) {
	logger.Infof("recv getRouterRtpCapabilities message")
	response := &WsMessage{
		Id:       req.Id,
		Response: true,
		Ok:       true,
		Data:     gConfig,
	}

	u.sendMsg <- response
}

func (u *User) handleCreateWebrtcTransport(req *WsMessage) {
	logger.Infof("recv createWebRtcTransport message")
	response := &WsMessage{
		Id:       req.Id,
		Response: true,
		Ok:       false,
	}
	room := gRoomManager.GetOrCreateRoom(u.roomId)
	if room != nil {
		reqDataBytes, err := json.Marshal(req.Data)
		if err == nil {
			var reqData CreateTransportReqData
			err := json.Unmarshal(reqDataBytes, &reqData)
			if err == nil {
				resData, err := room.CreateWebrtcTransport(&reqData, u)
				if err == nil {
					response.Ok = true
					response.Data = resData
				} else {
					logger.Errorf("create webrtc transport failed, reason=%v", err)
				}
			}
		}
	}
	u.sendMsg <- response
}

func (u *User) handleConnectWebrtcTransport(req *WsMessage) {
	logger.Infof("recv connectWebRtcTransport message")
	response := &WsMessage{
		Id:       req.Id,
		Response: true,
		Ok:       false,
	}
	room := gRoomManager.GetOrCreateRoom(u.roomId)
	if room != nil {
		reqDataBytes, err := json.Marshal(req.Data)
		if err == nil {
			var reqData ConnectTransportReqData
			err := json.Unmarshal(reqDataBytes, &reqData)
			if err == nil {
				if !u.newDataConsumerReq {
					newDataConsumerReqData := room.CreateNewDataConsumer(u, reqData.TransportId)
					if newDataConsumerReqData != nil {
						req := &WsMessage{
							Request: true,
							Id:      u.reqId,
							Method:  "newDataConsumer",
							Data:    newDataConsumerReqData,
						}
						u.reqId++

						u.sendMsg <- req
						u.newDataConsumerReq = true
					}
				}
				err := room.ConnectWebrtcTransport(&reqData)
				if err == nil {
					response.Ok = true
					response.Data = &ProduceDataResData{}
				} else {
					logger.Errorf("connect webrtc transport failed, reason=%v", err)
				}
			} else {
				logger.Errorf("transform ConnectTransportReqData failed, reason=%v", err)
			}

		}
	}

	u.sendMsg <- response
}

func (u *User) handleJoin(req *WsMessage) {
	logger.Infof("recv join message")
	room := gRoomManager.GetOrCreateRoom(u.roomId)
	if room != nil {
		reqDataBytes, err := json.Marshal(req.Data)
		if err == nil {
			var reqData JoinReqData
			err := json.Unmarshal(reqDataBytes, &reqData)
			if err == nil {
				u.Device = reqData.Device
				u.DisplayName = reqData.DisplayName
			} else {
				logger.Errorf("transform ProduceDataReqData failed, reason=%v", err)
			}
		}
		//notify other users
		room.NotifyOtherUsers(u)
	}

	otherUsers := room.GetOtherUsers(u)
	response := &WsMessage{
		Id:       req.Id,
		Response: true,
		Ok:       true,
		Data: &JoinResData{
			Peers: otherUsers,
		},
	}

	u.sendMsg <- response
}

func (u *User) handleProduceData(req *WsMessage) {
	logger.Infof("recv produceData message")
	response := &WsMessage{
		Id:       req.Id,
		Response: true,
		Ok:       false,
	}

	room := gRoomManager.GetOrCreateRoom(u.roomId)
	if room != nil {
		reqDataBytes, err := json.Marshal(req.Data)
		if err == nil {
			var reqData ProduceDataReqData
			err := json.Unmarshal(reqDataBytes, &reqData)
			if err == nil {
				response.Ok = true
				response.Data = &ProduceDataResData{
					DataProducerId: RandString(12),
				}
			} else {
				logger.Errorf("transform ProduceDataReqData failed, reason=%v", err)
			}

		}
	}

	u.sendMsg <- response
}

func (u *User) handleProduce(req *WsMessage) {
	logger.Infof("recv produce message")
	response := &WsMessage{
		Id:       req.Id,
		Response: true,
		Ok:       true,
	}

	room := gRoomManager.GetOrCreateRoom(u.roomId)
	if room != nil {
		reqDataBytes, err := json.Marshal(req.Data)
		if err == nil {
			var reqData ProduceReqData
			err := json.Unmarshal(reqDataBytes, &reqData)
			if err == nil {
				producerId, err := room.Produce(u, &reqData)
				if err == nil {
					response.Ok = true
					response.Data = &ProduceResData{
						ProducerId: producerId,
					}
				} else {
					logger.Errorf("produce req failed, reason=%v", err)
				}
			} else {
				logger.Errorf("transform ProduceReqData failed, reason=%v", err)
			}
		}
	}

	u.sendMsg <- response
}

func (u *User) HandleCloseProducer(notify *WsMessage) {
	room := gRoomManager.GetOrCreateRoom(u.roomId)
	if room != nil {
		reqDataBytes, err := json.Marshal(notify.Data)
		if err == nil {
			var reqData ProduceResData
			err := json.Unmarshal(reqDataBytes, &reqData)
			if err == nil {
				err := room.CloseProducer(u, reqData.ProducerId)
				if err == nil {
					logger.Infof("close producer successfully, userId=%s, producerId=%s", u.userId, reqData.ProducerId)
				} else {
					logger.Errorf("close producer failed, reason=%v", err)
				}
			} else {
				logger.Errorf("transform ProduceReqData failed, reason=%v", err)
			}
		}
	}
}

func (u *User) NotifyNewPeer(peerData *PeerData) {
	notify := &WsMessage{
		Notification: true,
		Method:       "newPeer",
		Data:         peerData,
	}

	u.sendMsg <- notify
}

func (u *User) NotifyPeerClosed(peer *Peer) {
	notify := &WsMessage{
		Notification: true,
		Method:       "peerClosed",
		Data:         peer,
	}

	u.sendMsg <- notify
}

func (u *User) RequestNewDataConsumer(reqData *NewDataConsumerReqData) {
	req := &WsMessage{
		Request: true,
		Id:      u.reqId,
		Method:  "newDataConsumer",
		Data:    reqData,
	}
	u.reqId++

	u.sendMsg <- req
}

func (u *User) RequestNewConsumer(reqData *NewConsumerReqData) {
	req := &WsMessage{
		Request: true,
		Id:      u.reqId,
		Method:  "newConsumer",
		Data:    reqData,
	}
	u.reqId++

	u.sendMsg <- req
}

func (u *User) NotifyConsumerClosed(notifyData *ConsumeResData) {
	notify := &WsMessage{
		Notification: true,
		Method:       "consumerClosed",
		Data:         notifyData,
	}

	u.sendMsg <- notify
}

func (u *User) WriteMessage() {
	defer func() {
		u.wsConn.Close()
		logger.Infof("close connection from write")
	}()

	for res := range u.sendMsg {
		jsonData, err := json.Marshal(res)
		if err != nil {
			logger.Info("failed to transform wsResponse to json")
			continue
		}
		logger.Infof("send response=%s to client", jsonData)

		if err := u.wsConn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			logger.Infof("write message failed: %v", err)
			return
		}
	}
}
