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
	userId             string
	PeerId             string
	DisplayName        string
	createTs           int64
	wsConn             *websocket.Conn
	wsServer           *WsServer
	sendResMsg         chan *WsResponse
	sendReqMsg         chan *WsRequest
	sendNotifyMsg      chan *WsNotification
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
			logger.Infof("unknown message type:=%s", wsReq.Method)
		}
	}
}

func (u *User) handleGetRouterRtpCapabilities(req *WsRequest) {
	logger.Infof("recv getRouterRtpCapabilities message")
	//str := `{"routerRtpCapabilities":{"codecs":[{"kind":"audio","mimeType":"audio/opus","clockRate":48000,"channels":2,"rtcpFeedback":[{"type":"nack","parameter":""},{"type":"transport-cc","parameter":""}],"parameters":{},"preferredPayloadType":100},{"kind":"video","mimeType":"video/VP8","clockRate":90000,"rtcpFeedback":[{"type":"nack","parameter":""},{"type":"nack","parameter":"pli"},{"type":"ccm","parameter":"fir"},{"type":"goog-remb","parameter":""},{"type":"transport-cc","parameter":""}],"parameters":{"x-google-start-bitrate":1000},"preferredPayloadType":101},{"kind":"video","mimeType":"video/rtx","preferredPayloadType":102,"clockRate":90000,"parameters":{"apt":101},"rtcpFeedback":[]},{"kind":"video","mimeType":"video/VP9","clockRate":90000,"rtcpFeedback":[{"type":"nack","parameter":""},{"type":"nack","parameter":"pli"},{"type":"ccm","parameter":"fir"},{"type":"goog-remb","parameter":""},{"type":"transport-cc","parameter":""}],"parameters":{"profile-id":2,"x-google-start-bitrate":1000},"preferredPayloadType":103},{"kind":"video","mimeType":"video/rtx","preferredPayloadType":104,"clockRate":90000,"parameters":{"apt":103},"rtcpFeedback":[]},{"kind":"video","mimeType":"video/H264","clockRate":90000,"parameters":{"level-asymmetry-allowed":1,"packetization-mode":1,"profile-level-id":"4d0032","x-google-start-bitrate":1000},"rtcpFeedback":[{"type":"nack","parameter":""},{"type":"nack","parameter":"pli"},{"type":"ccm","parameter":"fir"},{"type":"goog-remb","parameter":""},{"type":"transport-cc","parameter":""}],"preferredPayloadType":105},{"kind":"video","mimeType":"video/rtx","preferredPayloadType":106,"clockRate":90000,"parameters":{"apt":105},"rtcpFeedback":[]},{"kind":"video","mimeType":"video/H264","clockRate":90000,"parameters":{"level-asymmetry-allowed":1,"packetization-mode":1,"profile-level-id":"42e01f","x-google-start-bitrate":1000},"rtcpFeedback":[{"type":"nack","parameter":""},{"type":"nack","parameter":"pli"},{"type":"ccm","parameter":"fir"},{"type":"goog-remb","parameter":""},{"type":"transport-cc","parameter":""}],"preferredPayloadType":107},{"kind":"video","mimeType":"video/rtx","preferredPayloadType":108,"clockRate":90000,"parameters":{"apt":107},"rtcpFeedback":[]}],"headerExtensions":[{"kind":"audio","uri":"urn:ietf:params:rtp-hdrext:sdes:mid","preferredId":1,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"video","uri":"urn:ietf:params:rtp-hdrext:sdes:mid","preferredId":1,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"video","uri":"urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id","preferredId":2,"preferredEncrypt":false,"direction":"recvonly"},{"kind":"video","uri":"urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id","preferredId":3,"preferredEncrypt":false,"direction":"recvonly"},{"kind":"audio","uri":"http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time","preferredId":4,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"video","uri":"http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time","preferredId":4,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"audio","uri":"http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01","preferredId":5,"preferredEncrypt":false,"direction":"recvonly"},{"kind":"video","uri":"http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01","preferredId":5,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"audio","uri":"urn:ietf:params:rtp-hdrext:ssrc-audio-level","preferredId":6,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"video","uri":"https://aomediacodec.github.io/av1-rtp-spec/#dependency-descriptor-rtp-header-extension","preferredId":7,"preferredEncrypt":false,"direction":"recvonly"},{"kind":"video","uri":"urn:3gpp:video-orientation","preferredId":8,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"video","uri":"urn:ietf:params:rtp-hdrext:toffset","preferredId":9,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"audio","uri":"http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time","preferredId":10,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"video","uri":"http://www.webrtc.org/experiments/rtp-hdrext/abs-capture-time","preferredId":10,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"audio","uri":"http://www.webrtc.org/experiments/rtp-hdrext/playout-delay","preferredId":11,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"video","uri":"http://www.webrtc.org/experiments/rtp-hdrext/playout-delay","preferredId":11,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"audio","uri":"urn:mediasoup:params:rtp-hdrext:packet-id","preferredId":12,"preferredEncrypt":false,"direction":"sendrecv"},{"kind":"video","uri":"urn:mediasoup:params:rtp-hdrext:packet-id","preferredId":12,"preferredEncrypt":false,"direction":"sendrecv"}]}}`
	response := &WsResponse{
		Id:       req.Id,
		Response: true,
		Ok:       true,
		Data:     gConfig,
	}
	//json.Unmarshal([]byte(str), &response.Data)

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
				if !u.newDataConsumerReq {
					newDataConsumerReqData := room.CreateNewDataConsumer(u, reqData.TransportId)
					if newDataConsumerReqData != nil {
						req := &WsRequest{
							Request: true,
							Id:      u.reqId,
							Method:  "newDataConsumer",
							Data:    newDataConsumerReqData,
						}
						u.reqId++

						u.sendReqMsg <- req
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

	u.sendResMsg <- response
}

func (u *User) handleJoin(req *WsRequest) {
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
		//room.ReqOtherNewDataConsumer(u)
		//room.ReqOtherNewConsumer(u)
	}

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

func (u *User) NotifyPeerClosed(peer *Peer) {
	notify := &WsNotification{
		Notification: true,
		Method:       "peerClosed",
		Data:         peer,
	}

	u.sendNotifyMsg <- notify
}

func (u *User) RequestNewDataConsumer(reqData *NewDataConsumerReqData) {
	req := &WsRequest{
		Request: true,
		Id:      u.reqId,
		Method:  "newDataConsumer",
		Data:    reqData,
	}
	u.reqId++

	u.sendReqMsg <- req
}

func (u *User) RequestNewConsumer(reqData *NewConsumerReqData) {
	req := &WsRequest{
		Request: true,
		Id:      u.reqId,
		Method:  "newConsumer",
		Data:    reqData,
	}
	u.reqId++

	u.sendReqMsg <- req
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
			logger.Infof("send notification=%s to client", jsonData)

			if err := u.wsConn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				logger.Infof("write message failed: %v", err)
				return
			}
		}
	}

}
