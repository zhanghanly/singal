package singal

import (
	"encoding/json"

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
	Ok       bool        `json:"ok"`
	Data     interface{} `json:"data"`
}

type WsNotification struct {
	Notification bool        `json:"notification"`
	Method       string      `json:"method"`
	Data         interface{} `json:"data"`
}

type User struct {
	userId        string
	PeerId        string
	DisplayName   string
	createTs      int64
	wsConn        *websocket.Conn
	wsServer      *WsServer
	sendResMsg    chan *WsResponse
	sendReqMsg    chan *WsRequest
	sendNotifyMsg chan *WsNotification
	roomId        string
	Device        Device
	RemoteAddress string
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

		default:
			logger.Infof("unknown message type:")
		}
	}
}

func (u *User) handleGetRouterRtpCapabilities(req *WsRequest) {
	logger.Infof("recv getRouterRtpCapabilities message")
	response := &WsResponse{
		Id:       req.Id,
		Response: true,
		Ok:       true,
		Data:     gConfig,
	}
	u.sendResMsg <- response
}

func (u *User) handleCreateWebrtcTransport(req *WsRequest) {
	logger.Infof("recv createWebRtcTransport message")
	response := &WsResponse{
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
	u.sendResMsg <- response
}

func (u *User) handleConnectWebrtcTransport(req *WsRequest) {
	logger.Infof("recv connectWebRtcTransport message")
	response := &WsResponse{
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
				newDataConsumerReqData := room.CreateNewDataConsumer(u, reqData.TransportId)
				if newDataConsumerReqData != nil {
					req := &WsRequest{
						Request: true,
						Id:      10086,
						Method:  "newDataConsumer",
						Data:    newDataConsumerReqData,
					}

					u.sendReqMsg <- req
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

	u.sendResMsg <- response
}

func (u *User) handleJoin(req *WsRequest) {
	logger.Infof("recv join message")
	room := gRoomManager.GetOrCreateRoom(u.roomId)
	room.NotifyOtherUsers(u)

	otherUsers := room.GetOtherUsers(u)
	response := &WsResponse{
		Id:       req.Id,
		Response: true,
		Ok:       true,
		Data: &JoinResData{
			Peers: otherUsers,
		},
	}

	u.sendResMsg <- response
}

func (u *User) handleProduceData(req *WsRequest) {
	logger.Infof("recv produceData message")
	response := &WsResponse{
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
				//err := room.ConnectWebrtcTransport(&reqData)
				//if err == nil {
				response.Ok = true
				//} else {
				//	logger.Errorf("produce data req failed, reason=%v", err)
				//}
				response.Data = &ProduceDataResData{
					DataProducerId: RandString(12),
				}
			} else {
				logger.Errorf("transform ProduceDataReqData failed, reason=%v", err)
			}

		}
	}

	u.sendResMsg <- response
}

func (u *User) handleProduce(req *WsRequest) {
	logger.Infof("recv produce message")
	response := &WsResponse{
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
				err := room.Produce(&reqData)
				if err == nil {
					response.Ok = true
					response.Data = &ProduceResData{
						ProducerId: RandString(12),
					}
				} else {
					logger.Errorf("produce req failed, reason=%v", err)
				}
			} else {
				logger.Errorf("transform ProduceReqData failed, reason=%v", err)
			}
		}
	}

	u.sendResMsg <- response
}

func (u *User) NotifyNewPeer(peerData *PeerData) {
	notify := &WsNotification{
		Notification: true,
		Method:       "newPeer",
		Data:         peerData,
	}

	u.sendNotifyMsg <- notify
}

func (u *User) WriteMessage() {
	defer func() {
		u.wsConn.Close()
		logger.Infof("close connection from write")
	}()

	for {
		select {
		case res := <-u.sendResMsg:
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

		case req := <-u.sendReqMsg:
			jsonData, err := json.Marshal(req)
			if err != nil {
				logger.Info("failed to transform wsRequest to json")
				continue
			}
			logger.Infof("send request=%s to client", jsonData)

			if err := u.wsConn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				logger.Infof("write message failed: %v", err)
				return
			}

		case notify := <-u.sendNotifyMsg:
			jsonData, err := json.Marshal(notify)
			if err != nil {
				logger.Info("failed to transform wsNotification to json")
				continue
			}
			logger.Infof("send request=%s to client", jsonData)

			if err := u.wsConn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				logger.Infof("write message failed: %v", err)
				return
			}
		}
	}

}
