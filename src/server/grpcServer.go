package singal

import (
	"errors"
	"io"
	"math/rand"
	"net"
	pb "singal/src/server/proto"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var gRtcServer *WebRtcServer

type WorkerStream struct {
	stream    pb.WebRtcService_SyncServer
	pending   map[uint64]chan *pb.WorkerToServer
	mu        sync.Mutex
	worker    *Worker
	workerMgr *WorkerManager
}

type WebRtcServer struct {
	pb.UnimplementedWebRtcServiceServer
	workers sync.Map
	nextID  uint64
}

func NewWebRtcServer() *WebRtcServer {
	InitWorkerManager()
	return &WebRtcServer{
		workers: sync.Map{},
	}
}

func (s *WebRtcServer) Sync(stream pb.WebRtcService_SyncServer) error {
	workerID := ""
	var ws *WorkerStream

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				if workerID != "" {
					GetWorkerManager().RemoveWorker(workerID)
					s.workers.Delete(workerID)
					logger.Infof("Worker %s stream ended", workerID)
				}
				return nil
			}
			logger.Errorf("Error receiving from worker: %v", err)
			if workerID != "" {
				GetWorkerManager().RemoveWorker(workerID)
				s.workers.Delete(workerID)
			}
			return err
		}

		switch payload := msg.Payload.(type) {
		case *pb.WorkerToServer_WorkerRegister:
			if workerID != "" {
				GetWorkerManager().RemoveWorker(workerID)
				s.workers.Delete(workerID)
			}
			workerID = payload.WorkerRegister.WorkerId
			wm := GetWorkerManager()
			worker, err := wm.RegisterWorker(
				payload.WorkerRegister.WorkerId,
				payload.WorkerRegister.PublicIp,
				payload.WorkerRegister.PublicPort,
				payload.WorkerRegister.UseUdp,
			)
			if err != nil {
				logger.Errorf("Failed to register worker: %v", err)
				return err
			}

			ws = &WorkerStream{
				stream:    stream,
				pending:   make(map[uint64]chan *pb.WorkerToServer),
				worker:    worker,
				workerMgr: wm,
			}
			s.workers.Store(workerID, ws)

			resp := &pb.ServerToWorker{
				SeqId: msg.SeqId,
				Payload: &pb.ServerToWorker_WorkerRegisterRes{
					WorkerRegisterRes: &pb.WorkerRegisterResponse{
						WorkerId:    workerID,
						Success:     true,
						ErrorDetail: "",
					},
				},
			}
			if err := stream.Send(resp); err != nil {
				logger.Errorf("Failed to send register response: %v", err)
				return err
			}
			logger.Infof("Worker %s registered successfully", workerID)

		case *pb.WorkerToServer_WorkerKeepalive:
			logger.Info("recieve keepalive message")
			if payload.WorkerKeepalive == nil {
				continue
			}
			workerID = payload.WorkerKeepalive.WorkerId
			wm := GetWorkerManager()
			success := wm.Heartbeat(workerID)
			if success {
				if ws != nil {
					ws.workerMgr.UpdateWorkerStats(
						workerID,
						payload.WorkerKeepalive.RouterCount,
						payload.WorkerKeepalive.CpuUsage,
						payload.WorkerKeepalive.MemoryUsage,
					)
				}
			}

			resp := &pb.ServerToWorker{
				SeqId: msg.SeqId,
				Payload: &pb.ServerToWorker_WorkerKeepaliveRes{
					WorkerKeepaliveRes: &pb.WorkerKeepaliveResponse{
						Success: success,
					},
				},
			}
			if ws != nil {
				if err := ws.stream.Send(resp); err != nil {
					logger.Errorf("Failed to send heartbeat response: %v", err)
				}
			}

		case *pb.WorkerToServer_CreateRouterRes:
			if ws != nil {
				ws.mu.Lock()
				if ch, ok := ws.pending[msg.SeqId]; ok {
					ch <- msg
					delete(ws.pending, msg.SeqId)
				}
				ws.mu.Unlock()
			}

		case *pb.WorkerToServer_CreateTransportRes:
			if ws != nil {
				ws.mu.Lock()
				if ch, ok := ws.pending[msg.SeqId]; ok {
					ch <- msg
					delete(ws.pending, msg.SeqId)
				}
				ws.mu.Unlock()
			}

		case *pb.WorkerToServer_ConnectTransportRes:
			if ws != nil {
				ws.mu.Lock()
				if ch, ok := ws.pending[msg.SeqId]; ok {
					ch <- msg
					delete(ws.pending, msg.SeqId)
				}
				ws.mu.Unlock()
			}

		case *pb.WorkerToServer_ProduceRes:
			if ws != nil {
				ws.mu.Lock()
				if ch, ok := ws.pending[msg.SeqId]; ok {
					ch <- msg
					delete(ws.pending, msg.SeqId)
				}
				ws.mu.Unlock()
			}

		case *pb.WorkerToServer_ConsumerRes:
			if ws != nil {
				ws.mu.Lock()
				if ch, ok := ws.pending[msg.SeqId]; ok {
					ch <- msg
					delete(ws.pending, msg.SeqId)
				}
				ws.mu.Unlock()
			}
		}
	}
}

func (s *WebRtcServer) getWorkerStream(workerId string) (*WorkerStream, error) {
	val, ok := s.workers.Load(workerId)
	if !ok {
		return nil, errors.New("worker offline")
	}
	return val.(*WorkerStream), nil
}

func (s *WebRtcServer) sendRequest(workerId string, msg *pb.ServerToWorker) (*pb.WorkerToServer, error) {
	ws, err := s.getWorkerStream(workerId)
	if err != nil {
		return nil, err
	}

	seqID := atomic.AddUint64(&s.nextID, 1)
	msg.SeqId = seqID

	resCh := make(chan *pb.WorkerToServer, 1)

	ws.mu.Lock()
	ws.pending[seqID] = resCh
	ws.mu.Unlock()

	if err := ws.stream.Send(msg); err != nil {
		ws.mu.Lock()
		delete(ws.pending, seqID)
		ws.mu.Unlock()
		return nil, err
	}

	select {
	case resMsg := <-resCh:
		return resMsg, nil
	case <-time.After(10 * time.Second):
		ws.mu.Lock()
		delete(ws.pending, seqID)
		ws.mu.Unlock()
		return nil, errors.New("request timeout")
	}
}

func (s *WebRtcServer) CreateRouterOnWorker() (*Router, error) {
	worker := GetWorkerManager().ChooseBestWorker()
	if worker == nil {
		return nil, errors.New("no available worker")
	}

	router := &Router{
		routerId:            RandString(36),
		publicIp:            worker.publicIp,
		port:                worker.publicPort,
		preferenceUdp:       worker.useUdp,
		sendBufSize:         1024,
		recvBufSize:         1024,
		workerId:            worker.workerId,
		producers:           make(map[string][]*Producer),
		consumers:           make(map[string][]*Consumer),
		produceTransportIds: make(map[string]string),
		consumeTransportIds: make(map[string]string),
	}

	serverMsg := &pb.ServerToWorker{
		Payload: &pb.ServerToWorker_CreateRouterReq{
			CreateRouterReq: &pb.CreateRouterRequest{
				WorkerId: worker.workerId,
				RoomId:   router.routerId,
				ServerId: RandString(10),
				Info: &pb.ListenInfo{
					Protocol:         "UDP",
					Ip:               "0.0.0.0",
					Port:             router.port,
					AnnouncedAddress: router.publicIp,
					AnnouncedIp:      router.publicIp,
					AnnouncedPort:    router.port,
					Tcp:              false,
					Ipv6Only:         false,
					UdpReusePort:     false,
					RecvBufferSize:   router.recvBufSize,
					SendBufferSize:   router.sendBufSize,
				},
			},
		},
	}

	resMsg, err := s.sendRequest(worker.workerId, serverMsg)
	if err != nil {
		return nil, err
	}

	res := resMsg.GetCreateRouterRes()
	if !res.Success {
		return nil, errors.New(res.ErrorDetail)
	}

	GetWorkerManager().AddRouter(worker.workerId, router)
	logger.Infof("Router created on worker %s: routerId=%s", worker.workerId, router.routerId)

	return router, nil
}

func (s *WebRtcServer) CreateWebrtcTransport(router *Router) (*pb.CreateTransportResponse, error) {
	if router == nil || router.workerId == "" {
		return nil, errors.New("invalid router")
	}

	serverMsg := &pb.ServerToWorker{
		Payload: &pb.ServerToWorker_CreateTransportReq{
			CreateTransportReq: &pb.CreateTransportRequest{
				WorkerId:    router.workerId,
				RouterId:    router.routerId,
				TransportId: RandString(36),
			},
		},
	}

	resMsg, err := s.sendRequest(router.workerId, serverMsg)
	if err != nil {
		return nil, err
	}

	res := resMsg.GetCreateTransportRes()
	if !res.Success {
		return nil, errors.New(res.ErrorDetail)
	}

	return res, nil
}

func (s *WebRtcServer) ConnectWebrtcTransport(router *Router, req *ConnectTransportReqData) error {
	if router == nil || router.workerId == "" {
		return errors.New("invalid router")
	}

	dtlsFingerprints := make([]*pb.DtlsFingerprint, 0)
	for _, v := range req.DTLSParameters.Fingerprints {
		fingerprint := &pb.DtlsFingerprint{
			Algorithm: v.Algorithm,
			Value:     v.Value,
		}
		dtlsFingerprints = append(dtlsFingerprints, fingerprint)
	}

	serverMsg := &pb.ServerToWorker{
		Payload: &pb.ServerToWorker_ConnectTransportReq{
			ConnectTransportReq: &pb.ConnectTransportRequest{
				WorkerId:         router.workerId,
				RouterId:         router.routerId,
				TransportId:      req.TransportId,
				DtlsRole:         req.DTLSParameters.Role,
				DtlsFingerprints: dtlsFingerprints,
			},
		},
	}

	resMsg, err := s.sendRequest(router.workerId, serverMsg)
	if err != nil {
		return err
	}

	res := resMsg.GetConnectTransportRes()
	if !res.Success {
		return errors.New(res.ErrorDetail)
	}

	return nil
}

func (s *WebRtcServer) CreateProducer(router *Router, producer *Producer) (*pb.ProduceResponse, error) {
	if router == nil || router.workerId == "" {
		return nil, errors.New("invalid router")
	}

	codecs := make([]*pb.Codec, 0)
	for _, v := range producer.parameters.RtpParameters.MediaCodecs {
		rtcpFeedbacks := make([]*pb.RtcpFeedback, 0)
		for _, feedback := range v.Feedbacks {
			if producer.kind == "audio" && feedback.Type == "nack" {
				continue
			}

			rtcpFeedback := &pb.RtcpFeedback{
				Type:      feedback.Type,
				Parameter: feedback.Parameter,
			}
			rtcpFeedbacks = append(rtcpFeedbacks, rtcpFeedback)
		}

		codec := &pb.Codec{
			MimeType:      v.MimeType,
			PayloadType:   uint32(v.PayloadType),
			ClockRate:     uint32(v.ClockRate),
			Channels:      uint32(v.Channels),
			RtcpFeedbacks: rtcpFeedbacks,
		}

		codecs = append(codecs, codec)
	}

	headerExtensions := make([]*pb.HeadExtension, 0)
	for _, v := range producer.parameters.RtpParameters.HeaderExtensions {
		headerExtension := &pb.HeadExtension{
			Uri:     v.Uri,
			Id:      uint32(v.Id),
			Encrypt: v.Encrypt,
		}
		headerExtensions = append(headerExtensions, headerExtension)
	}

	encodings := make([]*pb.Encoding, 0)
	for _, v := range producer.parameters.RtpParameters.Encodings {
		encoding := &pb.Encoding{
			Active:                v.Active,
			ScalabilityMode:       v.ScalabilityMode,
			ScaleResolutionDownBy: uint32(v.ScaleResolutionDownBy),
			MaxBitrate:            uint32(v.MaxBitrate),
			Rid:                   v.Rid,
			Dtx:                   v.Dtx,
			Ssrc:                  v.Ssrc,
		}
		if producer.kind == "video" {
			if v.Rtx == nil {
				ssrc := rand.Uint32()
				v.Ssrc = ssrc
				v.Rtx = &Rtx{
					Ssrc: ssrc + 1,
				}
			}
			encoding.Ssrc = v.Ssrc
			encoding.RtxSsrc = v.Rtx.Ssrc
			encoding.HasRtx = true
		}

		encodings = append(encodings, encoding)
	}

	rtpMappings := &pb.RtpMapping{
		PayloadMap:  make([]*pb.PayloadMap, 0),
		EncodingMap: make([]*pb.EncodingMap, 0),
	}
	for _, v := range encodings {
		encodingMap := &pb.EncodingMap{
			Rid:        v.Rid,
			Ssrc:       v.Ssrc,
			MappedSsrc: v.Ssrc,
		}
		rtpMappings.EncodingMap = append(rtpMappings.EncodingMap, encodingMap)
	}

	for _, v := range codecs {
		payloadMap := &pb.PayloadMap{}
		if producer.kind == "video" {
			payloadMap.PayloadType = v.PayloadType
			payloadMap.MappedPayloadType = v.PayloadType + 5
		} else {
			payloadMap.PayloadType = v.PayloadType
			payloadMap.MappedPayloadType = v.PayloadType - 11
		}
		rtpMappings.PayloadMap = append(rtpMappings.PayloadMap, payloadMap)
	}

	serverMsg := &pb.ServerToWorker{
		Payload: &pb.ServerToWorker_ProduceReq{
			ProduceReq: &pb.ProduceRequest{
				RouterId:    router.routerId,
				TransportId: producer.transportId,
				ProducerId:  producer.producerId,
				Kind:        producer.kind,
				Method:      "create",
				RtpParameters: &pb.RtpParameters{
					Mid:  producer.parameters.RtpParameters.Mid,
					Msid: producer.parameters.RtpParameters.Msid,
					Rtcp: &pb.Rtcp{
						Cname:       producer.parameters.RtpParameters.Rtcp.CName,
						ReducedSize: producer.parameters.RtpParameters.Rtcp.ReducedSize,
					},
					Codecs:         codecs,
					HeadExtensions: headerExtensions,
					Encodings:      encodings,
					RtpMapping:     rtpMappings,
				},
			},
		},
	}

	resMsg, err := s.sendRequest(router.workerId, serverMsg)
	if err != nil {
		return nil, err
	}

	res := resMsg.GetProduceRes()
	if !res.Success {
		return nil, errors.New(res.ErrorDetail)
	}

	return res, nil
}

func (s *WebRtcServer) CloseProducer(router *Router, producer *Producer) (*pb.ProduceResponse, error) {
	if router == nil || router.workerId == "" {
		return nil, errors.New("invalid router")
	}
	serverMsg := &pb.ServerToWorker{
		Payload: &pb.ServerToWorker_ProduceReq{
			ProduceReq: &pb.ProduceRequest{
				RouterId:    router.routerId,
				TransportId: producer.transportId,
				ProducerId:  producer.producerId,
				Kind:        producer.kind,
				Method:      "close",
			},
		},
	}

	resMsg, err := s.sendRequest(router.workerId, serverMsg)
	if err != nil {
		return nil, err
	}

	res := resMsg.GetProduceRes()
	if !res.Success {
		return nil, errors.New(res.ErrorDetail)
	}

	return res, nil
}

func (s *WebRtcServer) CreateConsumer(router *Router, consumerReq *NewConsumerReqData) (*pb.ConsumeResponse, error) {
	if router == nil || router.workerId == "" {
		return nil, errors.New("invalid router")
	}

	codecs := make([]*pb.Codec, 0)
	for _, v := range consumerReq.RtpParameters.MediaCodecs {
		rtcpFeedbacks := make([]*pb.RtcpFeedback, 0)
		for _, feedback := range v.Feedbacks {
			rtcpFeedback := &pb.RtcpFeedback{
				Type:      feedback.Type,
				Parameter: feedback.Parameter,
			}
			rtcpFeedbacks = append(rtcpFeedbacks, rtcpFeedback)
		}

		codec := &pb.Codec{
			MimeType:      v.MimeType,
			PayloadType:   uint32(v.PayloadType),
			ClockRate:     uint32(v.ClockRate),
			Channels:      uint32(v.Channels),
			RtcpFeedbacks: rtcpFeedbacks,
		}

		codecs = append(codecs, codec)
	}

	headerExtensions := make([]*pb.HeadExtension, 0)
	for _, v := range consumerReq.RtpParameters.HeaderExtensions {
		headerExtension := &pb.HeadExtension{
			Uri:     v.Uri,
			Id:      uint32(v.Id),
			Encrypt: v.Encrypt,
		}
		headerExtensions = append(headerExtensions, headerExtension)
	}

	encodings := make([]*pb.Encoding, 0)
	for _, v := range consumerReq.RtpParameters.Encodings {
		encoding := &pb.Encoding{
			Active:                v.Active,
			ScalabilityMode:       v.ScalabilityMode,
			ScaleResolutionDownBy: uint32(v.ScaleResolutionDownBy),
			MaxBitrate:            uint32(v.MaxBitrate),
			Rid:                   v.Rid,
			Dtx:                   v.Dtx,
			Ssrc:                  v.Ssrc,
			HasRtx:                false,
		}
		if consumerReq.Kind == "video" {
			encoding.HasRtx = true
			encoding.RtxSsrc = v.Rtx.Ssrc
		}
		encodings = append(encodings, encoding)
	}

	rtpMappings := &pb.RtpMapping{
		PayloadMap:  make([]*pb.PayloadMap, 0),
		EncodingMap: make([]*pb.EncodingMap, 0),
	}
	for _, v := range encodings {
		encodingMap := &pb.EncodingMap{
			Rid:        v.Rid,
			Ssrc:       v.Ssrc,
			MappedSsrc: v.Ssrc,
		}
		rtpMappings.EncodingMap = append(rtpMappings.EncodingMap, encodingMap)
	}

	for _, v := range codecs {
		payloadMap := &pb.PayloadMap{}
		if consumerReq.Kind == "video" {
			payloadMap.PayloadType = v.PayloadType
			payloadMap.MappedPayloadType = v.PayloadType + 5
		} else {
			payloadMap.PayloadType = v.PayloadType
			payloadMap.MappedPayloadType = v.PayloadType - 11
		}
		rtpMappings.PayloadMap = append(rtpMappings.PayloadMap, payloadMap)
	}

	serverMsg := &pb.ServerToWorker{
		Payload: &pb.ServerToWorker_ConsumerReq{
			ConsumerReq: &pb.ConsumeRequest{
				RouterId:    router.routerId,
				TransportId: consumerReq.TransportId,
				ProducerId:  consumerReq.ProducerId,
				ConsumerId:  consumerReq.ConsumerId,
				Kind:        consumerReq.Kind,
				Method:      "create",
				RtpParameters: &pb.RtpParameters{
					Mid:  consumerReq.RtpParameters.Mid,
					Msid: consumerReq.RtpParameters.Msid,
					Rtcp: &pb.Rtcp{
						Cname:       consumerReq.RtpParameters.Rtcp.CName,
						ReducedSize: consumerReq.RtpParameters.Rtcp.ReducedSize,
					},
					Codecs:         codecs,
					HeadExtensions: headerExtensions,
					Encodings:      encodings,
					RtpMapping:     rtpMappings,
				},
			},
		},
	}

	resMsg, err := s.sendRequest(router.workerId, serverMsg)
	if err != nil {
		return nil, err
	}

	res := resMsg.GetConsumerRes()
	if !res.Success {
		return nil, errors.New(res.ErrorDetail)
	}

	return res, nil
}

func (s *WebRtcServer) CloseConsumer(router *Router, consumer *Consumer) (*pb.ConsumeResponse, error) {
	if router == nil || router.workerId == "" {
		return nil, errors.New("invalid router")
	}
	serverMsg := &pb.ServerToWorker{
		Payload: &pb.ServerToWorker_ConsumerReq{
			ConsumerReq: &pb.ConsumeRequest{
				RouterId:    router.routerId,
				TransportId: consumer.transportId,
				ProducerId:  consumer.producerId,
				ConsumerId:  consumer.consumerId,
				Kind:        consumer.kind,
				Method:      "close",
			},
		},
	}
	resMsg, err := s.sendRequest(router.workerId, serverMsg)
	if err != nil {
		return nil, err
	}

	res := resMsg.GetConsumerRes()
	if !res.Success {
		return nil, errors.New(res.ErrorDetail)
	}

	return res, nil
}

func StartGrpcServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Errorf("failed to listen: %v", err)
		return
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    15 * time.Second,
			Timeout: 50 * time.Second,
		}),
	)

	gRtcServer = NewWebRtcServer()
	pb.RegisterWebRtcServiceServer(grpcServer, gRtcServer)

	logger.Infoln("gRPC WebRTC Server running on :50051...")
	if err := grpcServer.Serve(lis); err != nil {
		logger.Errorf("failed to serve: %v", err)
	}
}
