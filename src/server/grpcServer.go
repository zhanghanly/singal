package singal

import (
	"errors"
	pb "singal/src/server/proto"
	"sync"
	"sync/atomic"
	"time"
)

type WorkerStream struct {
	stream  pb.WebRtcService_SyncServer
	pending map[uint64]chan *pb.WorkerToServer
	mu      sync.Mutex
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

func (s *WebRtcServer) CreateRouterOnWorker(workerID uint64, req *pb.CreateRouterRequest) (*pb.CreateRouterResponse, error) {
	val, ok := s.workers.Load(workerID)
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
		Payload: &pb.ServerToWorker_CreateRouterReq{
			CreateRouterReq: req,
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
		return res, nil

	case <-time.After(5 * time.Second):
		return nil, errors.New("request timeout")
	}
}
