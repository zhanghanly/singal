package singal

import (
	"google.golang.org/grpc"
	pb "singal/src/server/proto"
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

type Router struct {
	id        string
	producers map[string]*Producer
	consumers map[string]*Consumer
}

type SfuNode struct {
	id         string
	publicIp   string
	minPort    uint32
	maxPort    uint32
	lastActive int64
	routers    map[string]*Router
}

type GrpcServer struct {
}

func NewGrpcServer() *GrpcServer {

}
