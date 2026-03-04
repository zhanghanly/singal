package singal

import (
	"errors"
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
	stream  pb.WebRtcService_SyncServer
	pending map[uint64]chan *pb.WorkerToServer
	mu      sync.Mutex
	worker  *Worker
}

type WebRtcServer struct {
	pb.UnimplementedWebRtcServiceServer
	workers sync.Map // map[string]*WorkerStream
	nextID  uint64
}

func NewWebRtcServer() *WebRtcServer {
	return &WebRtcServer{}
}

func (s *WebRtcServer) Sync(stream pb.WebRtcService_SyncServer) error {
	logger.Infof("begin to recv grpc peer message")
	//firstReq, err := stream.Recv()
	_, err := stream.Recv()
	if err != nil {
		logger.Infof("recv grpc peer message failed, %v", err)
		return err
	}
	//workerID := firstReq.SeqId
	workerID := "1"

	ws := &WorkerStream{
		stream:  stream,
		pending: make(map[uint64]chan *pb.WorkerToServer),
		worker: &Worker{
			publicIp:      "172.20.10.8",
			inUsePort:     44444,
			preferenceUdp: true,
			routers:       make(map[string]*Router),
		},
	}
	s.workers.Store(workerID, ws)
	defer s.workers.Delete(workerID)

	logger.Infof("Worker %s connected", workerID)

	for {
		resp, err := stream.Recv()
		if err != nil {
			logger.Errorf("Worker %s disconnected: %v", workerID, err)
			return err
		}

		ws.mu.Lock()
		if ch, ok := ws.pending[resp.SeqId]; ok {
			ch <- resp
			delete(ws.pending, resp.SeqId)
		}
		ws.mu.Unlock()
	}
}

func (s *WebRtcServer) CreateRouterOnWorker() (*Router, error) {
	workerId := s.chooseBestWorker()
	val, ok := s.workers.Load(workerId)
	if !ok {
		return nil, errors.New("worker offline")
	}
	conn := val.(*WorkerStream)

	seqID := atomic.AddUint64(&s.nextID, 1)
	resCh := make(chan *pb.WorkerToServer, 1)

	conn.mu.Lock()
	conn.pending[seqID] = resCh
	conn.mu.Unlock()

	router := conn.worker.CreateRouter()
	serverMsg := &pb.ServerToWorker{
		SeqId: seqID,
		Payload: &pb.ServerToWorker_CreateRouterReq{
			CreateRouterReq: &pb.CreateRouterRequest{
				WorkerId: workerId,
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
	if err := conn.stream.Send(serverMsg); err != nil {
		return nil, err
	}

	// wait or timeout
	select {
	case resMsg := <-resCh:
		res := resMsg.GetCreateRouterRes()
		if !res.Success {
			return nil, errors.New(res.ErrorDetail)
		}

		conn.worker.AddRouter(router)
		return router, nil

	case <-time.After(5 * time.Second):
		return nil, errors.New("request timeout")
	}
}

func (s *WebRtcServer) CreateWebrtcTransport(router *Router) (*pb.CreateTransportResponse, error) {
	//workerId := router.workerId
	workerId := "1"
	val, ok := s.workers.Load(workerId)
	if !ok {
		return nil, errors.New("worker offline")
	}
	conn := val.(*WorkerStream)

	seqID := atomic.AddUint64(&s.nextID, 1)
	resCh := make(chan *pb.WorkerToServer, 1)

	conn.mu.Lock()
	conn.pending[seqID] = resCh
	conn.mu.Unlock()

	serverMsg := &pb.ServerToWorker{
		SeqId: seqID,
		Payload: &pb.ServerToWorker_CreateTransportReq{
			CreateTransportReq: &pb.CreateTransportRequest{
				WorkerId:    workerId,
				RouterId:    router.routerId,
				TransportId: RandString(36),
			},
		},
	}
	if err := conn.stream.Send(serverMsg); err != nil {
		return nil, err
	}

	// wait or timeout
	select {
	case resMsg := <-resCh:
		res := resMsg.GetCreateTransportRes()
		if !res.Success {
			return nil, errors.New(res.ErrorDetail)
		}

		return res, nil

	case <-time.After(5 * time.Second):
		return nil, errors.New("request timeout")
	}
}

func (s *WebRtcServer) ConnectWebrtcTransport(router *Router, req *ConnectTransportReqData) error {
	workerId := "1"
	val, ok := s.workers.Load(workerId)
	if !ok {
		return errors.New("worker offline")
	}
	conn := val.(*WorkerStream)

	seqID := atomic.AddUint64(&s.nextID, 1)
	resCh := make(chan *pb.WorkerToServer, 1)

	conn.mu.Lock()
	conn.pending[seqID] = resCh
	conn.mu.Unlock()

	dtlsFingerprints := make([]*pb.DtlsFingerprint, 0)
	for _, v := range req.DTLSParameters.Fingerprints {
		fingerprint := &pb.DtlsFingerprint{
			Algorithm: v.Algorithm,
			Value:     v.Value,
		}
		dtlsFingerprints = append(dtlsFingerprints, fingerprint)
	}
	serverMsg := &pb.ServerToWorker{
		SeqId: seqID,
		Payload: &pb.ServerToWorker_ConnectTransportReq{
			ConnectTransportReq: &pb.ConnectTransportRequest{
				WorkerId:         workerId,
				RouterId:         router.routerId,
				TransportId:      req.TransportId,
				DtlsRole:         req.DTLSParameters.Role,
				DtlsFingerprints: dtlsFingerprints,
			},
		},
	}

	if err := conn.stream.Send(serverMsg); err != nil {
		return err
	}

	// wait or timeout
	select {
	case resMsg := <-resCh:
		res := resMsg.GetConnectTransportRes()
		if !res.Success {
			return errors.New(res.ErrorDetail)
		}

		return nil

	case <-time.After(5 * time.Second):
		return errors.New("request timeout")
	}
}

func (s *WebRtcServer) CreateProducer(router *Router, producer *Producer) (*pb.ProduceResponse, error) {
	//workerId := router.workerId
	workerId := "1"
	val, ok := s.workers.Load(workerId)
	if !ok {
		return nil, errors.New("worker offline")
	}
	conn := val.(*WorkerStream)

	seqID := atomic.AddUint64(&s.nextID, 1)
	resCh := make(chan *pb.WorkerToServer, 1)

	conn.mu.Lock()
	conn.pending[seqID] = resCh
	conn.mu.Unlock()

	codecs := make([]*pb.Codec, 0)
	for _, v := range producer.parameters.RtpParameters.MediaCodecs {
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
			PayloadType:   uint32(v.PreferredPayloadType),
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

		encodings = append(encodings, encoding)
	}

	rtpMappings := make([]*pb.RtpMapping, 0)
	for _, v := range encodings {
		rtpMapping := &pb.RtpMapping{
			Rid:        v.Rid,
			Ssrc:       v.Ssrc,
			MappedSsrc: v.Ssrc,
		}

		rtpMappings = append(rtpMappings, rtpMapping)
	}

	for k, v := range codecs {
		rtpMappings[k].PayloadType = v.PayloadType
		rtpMappings[k].MappedPayloadType = v.PayloadType
	}

	serverMsg := &pb.ServerToWorker{
		SeqId: seqID,
		Payload: &pb.ServerToWorker_ProduceReq{
			ProduceReq: &pb.ProduceRequest{
				RouterId:    router.routerId,
				TransportId: producer.transportId,
				ProducerId:  producer.producerId,
				Kind:        producer.kind,
				RtpParameters: &pb.RtpParameters{
					Mid:  producer.parameters.RtpParameters.Mid,
					Msid: producer.parameters.Msid,
					Rtcp: &pb.Rtcp{
						Cname:       producer.parameters.Rtcp.CName,
						ReducedSize: producer.parameters.Rtcp.ReducedSize,
					},
					Codecs:         codecs,
					HeadExtensions: headerExtensions,
					Encodings:      encodings,
					RtpMapping:     rtpMappings,
				},
			},
		},
	}
	if err := conn.stream.Send(serverMsg); err != nil {
		return nil, err
	}

	// wait or timeout
	select {
	case resMsg := <-resCh:
		res := resMsg.GetProduceRes()
		if !res.Success {
			return nil, errors.New(res.ErrorDetail)
		}

		return res, nil

	case <-time.After(5 * time.Second):
		return nil, errors.New("request timeout")
	}
}

func (s *WebRtcServer) chooseBestWorker() string {
	return "1"
}

func StartGrpcServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Errorf("failed to listen: %v", err)
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
