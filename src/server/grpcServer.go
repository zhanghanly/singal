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
	firstReq, err := stream.Recv()
	if err != nil {
		return err
	}
	workerID := firstReq.SeqId

	ws := &WorkerStream{
		stream:  stream,
		pending: make(map[uint64]chan *pb.WorkerToServer),
	}
	s.workers.Store(workerID, ws)
	defer s.workers.Delete(workerID)

	logger.Infof("Worker %d connected", workerID)

	for {
		resp, err := stream.Recv()
		if err != nil {
			logger.Infof("Worker %d disconnected: %v", workerID, err)
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
					AnnouncedAddress: "",
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
	workerId := router.workerId
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
				TransportId: RandString(12),
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

func (s *WebRtcServer) chooseBestWorker() string {
	return "1"
}

func StartGrpcServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    15 * time.Second,
			Timeout: 5 * time.Second,
		}),
	)

	gRtcServer = NewWebRtcServer()
	pb.RegisterWebRtcServiceServer(grpcServer, gRtcServer)

	logger.Infoln("gRPC WebRTC Server running on :50051...")
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatalf("failed to serve: %v", err)
	}
}
