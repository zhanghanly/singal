package singal

import (
	"io"
	pb "singal/src/server/proto"
	"time"
)

type MediaType int32

const (
	VIDEO MediaType = 0
	AUDIO MediaType = 1
)

type Producer struct {
	id        string
	mediaType MediaType
	ssrc      uint32
}

type Consumer struct {
	id         string
	mediaType  MediaType
	originSsrc uint32
	newSsrc    uint32
}

type SfuNode struct {
	id        string
	publicIp  string
	minPort   uint32
	maxPort   uint32
	lastAlive int64
	stream    pb.SfuService_SfuSessionServer
}

func NewSfuNode(stream1 pb.SfuService_SfuSessionServer) *SfuNode {
	return &SfuNode{
		stream: stream1,
	}
}

type GrpcServer struct {
	pb.UnimplementedSfuServiceServer
	sfuNodes map[string]*SfuNode
}

func NewGrpcServer() *GrpcServer {
	return &GrpcServer{
		sfuNodes: make(map[string]*SfuNode),
	}
}

// SfuSession 双向流处理
func (g *GrpcServer) SfuSession(stream pb.SfuService_SfuSessionServer) error {
	logger.Infof("New game session stream established")

	var node *SfuNode
	defer func() {
		if node != nil {
			g.cleanSfuNode(node.id)
		}
	}()

	// 处理接收到的消息
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			logger.Fatal("Client disconnected:")
			return nil
		}
		if err != nil {
			logger.Fatal("Error receiving message:")
			return err
		}

		resp, shouldExit := g.handleClientMsg(stream, msg, &node)
		if shouldExit {
			return nil
		}

		// 发送响应
		if resp != nil {
			if err := stream.Send(resp); err != nil {
				logger.Fatal("Failed to send response: %v", err)
				return err
			}
		}
	}
}

func (g *GrpcServer) handleClientMsg(
	stream pb.SfuService_SfuSessionServer,
	msg *pb.SfuMessage,
	node **SfuNode) (*pb.SfuMessage, bool) {
	switch msg.Type {
	case pb.MessageType_REGISTER:
		return g.handleSfuNodeRegister(stream, msg, node)

	case pb.MessageType_KEEPALIVE:
		return g.handleSfuNodeKeepalive(stream, msg, node)

	case pb.MessageType_FLOW_REPORT:
		return g.handleSfuNodeFlowReport(stream, msg, node)

	case pb.MessageType_AUDIO_LEVEL_REPORT:
		return g.handleSfuNodeAudioLevel(stream, msg, node)

	default:
		logger.Warn("Unknown message type: %v", msg.Type)
		return &pb.SfuMessage{
			Type:    pb.MessageType_UNKNOWN,
			Content: &pb.SfuMessage_TextMessage{TextMessage: "Unknown message type"},
		}, false
	}
}

func (g *GrpcServer) handleSfuNodeRegister(
	stream pb.SfuService_SfuSessionServer,
	msg *pb.SfuMessage,
	node **SfuNode) (*pb.SfuMessage, bool) {
	regReq := msg.GetRegisterRequest()
	if _, ok := g.sfuNodes[regReq.ServerId]; !ok {
		sfuNode := NewSfuNode(stream)
		g.sfuNodes[regReq.ServerId] = sfuNode
	}

	return &pb.SfuMessage{
		Type:    pb.MessageType_UNKNOWN,
		Content: &pb.SfuMessage_TextMessage{TextMessage: "Unknown message type"},
	}, false
}

func (g *GrpcServer) handleSfuNodeKeepalive(
	stream pb.SfuService_SfuSessionServer,
	msg *pb.SfuMessage,
	node **SfuNode) (*pb.SfuMessage, bool) {
	aliveReq := msg.GetKeepaliveRequest()
	if sfuNode, ok := g.sfuNodes[aliveReq.ServerId]; ok {
		sfuNode.lastAlive = time.Now().Unix()
	}

	return &pb.SfuMessage{
		Type:    pb.MessageType_UNKNOWN,
		Content: &pb.SfuMessage_TextMessage{TextMessage: "Unknown message type"},
	}, false
}

func (g *GrpcServer) handleSfuNodeFlowReport(
	stream pb.SfuService_SfuSessionServer,
	msg *pb.SfuMessage,
	node **SfuNode) (*pb.SfuMessage, bool) {
	return &pb.SfuMessage{
		Type:    pb.MessageType_UNKNOWN,
		Content: &pb.SfuMessage_TextMessage{TextMessage: "Unknown message type"},
	}, false
}

func (g *GrpcServer) handleSfuNodeAudioLevel(
	stream pb.SfuService_SfuSessionServer,
	msg *pb.SfuMessage,
	node **SfuNode) (*pb.SfuMessage, bool) {
	return &pb.SfuMessage{
		Type:    pb.MessageType_UNKNOWN,
		Content: &pb.SfuMessage_TextMessage{TextMessage: "Unknown message type"},
	}, false
}

func (g *GrpcServer) cleanSfuNode(nodeId string) {
	delete(g.sfuNodes, nodeId)
}

func (g *GrpcServer) createRoom(room *Room) error {
	node := room.node
	if node != nil {
		req := &pb.SfuMessage{
			Type:      pb.MessageType_CREATE_ROUTER,
			Timestamp: time.Now().Unix(),
			Content: &pb.SfuMessage_CreateRouteRequest{
				&pb.CreateRouterRequest{
					ServerId: node.id,
				},
			},
		}
		if err := node.stream.Send(req); err != nil {
			logger.Fatal("Failed to send create room response: %v", err)
			return err
		}
	}

	return nil
}

func (g *GrpcServer) createProducer(user *User) error {
	node := user.node
	if node != nil {
		req := &pb.SfuMessage{
			Type:      pb.MessageType_PRODUCE_UPDATE,
			Timestamp: time.Now().Unix(),
			Content: &pb.SfuMessage_ProduceStateRequest{
				&pb.ProduceStateRequest{
					ServerId: node.id,
				},
			},
		}
		if err := node.stream.Send(req); err != nil {
			logger.Fatal("Failed to send create room response: %v", err)
			return err
		}
	}

	return nil
}

func (g *GrpcServer) createConsumer(user *User) error {
	node := user.node
	if node != nil {
		req := &pb.SfuMessage{
			Type:      pb.MessageType_CONSUME_UPDATE,
			Timestamp: time.Now().Unix(),
			Content: &pb.SfuMessage_ConsumeStateRequest{
				&pb.ConsumeStateRequest{
					ServerId: node.id,
				},
			},
		}
		if err := node.stream.Send(req); err != nil {
			logger.Fatal("Failed to send create room response: %v", err)
			return err
		}
	}

	return nil
}
