package singal

import (
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

type RtpHeaderExtension struct {
	uri        string
	id         uint32
	encrypt    bool
	parameters interface{}
}

type SfuNode struct {
	id        string
	publicIp  string
	minPort   uint32
	maxPort   uint32
	lastAlive int64
	stream    pb.SfuService_SfuSessionServer
}
