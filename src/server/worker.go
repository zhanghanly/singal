package singal

import (
	"strconv"
	"sync"
	"time"
)

type MediaType int32

const (
	VIDEO MediaType = 0
	AUDIO MediaType = 1
	DATA  MediaType = 2
)

type Producer struct {
	producerId  string
	transportId string
	kind        string
	ssrc        uint32
	parameters  *ProduceReqData
}

type Consumer struct {
	consumerId  string
	transportId string
	producerId  string
	kind        string
	originSsrc  uint32
	newSsrc     uint32
}

type RtpHeaderExtension struct {
	uri        string
	id         uint32
	encrypt    bool
	parameters interface{}
}

type Router struct {
	routerId      string
	publicIp      string
	minPort       uint32
	port          uint32
	maxPort       uint32
	preferenceUdp bool

	sendBufSize uint32
	recvBufSize uint32

	workerId string

	producers map[string][]*Producer
	consumers map[string][]*Consumer

	produceTransportIds map[string]string
	consumeTransportIds map[string]string
}

func (r *Router) addProducer(userId string, producer *Producer) {
	if producer != nil {
		if _, exist := r.producers[userId]; !exist {
			r.producers[userId] = make([]*Producer, 0)
		}

		r.producers[userId] = append(r.producers[userId], producer)
	}
}

func (r *Router) addConsumer(userId string, consumer *Consumer) {
	if consumer != nil {
		if _, exist := r.consumers[userId]; !exist {
			r.consumers[userId] = make([]*Consumer, 0)
		}

		r.consumers[userId] = append(r.consumers[userId], consumer)
	}
}

func (r *Router) CreateConsumes(userId string) {
	audioConsumer := &Consumer{
		consumerId:  RandString(36),
		transportId: r.getConsumerTransportId(userId),
		kind:        "audio",
	}
	r.addConsumer(userId, audioConsumer)

	videConsumer := &Consumer{
		consumerId:  RandString(36),
		transportId: r.getConsumerTransportId(userId),
		kind:        "video",
	}
	r.addConsumer(userId, videConsumer)
}

func (r *Router) getConsumerTransportId(userId string) string {
	if _, exist := r.consumeTransportIds[userId]; exist {
		return r.consumeTransportIds[userId]
	}

	return ""
}

func (r *Router) saveConsumeTransportId(userId string, transportId string) {
	if _, exist := r.consumeTransportIds[userId]; !exist {
		r.consumeTransportIds[userId] = transportId
		logger.Infof("save consume transport id=%s for user=%s",
			r.consumeTransportIds[userId], transportId)
	}
}

func (r *Router) saveProducerTransportId(userId string, transportId string) {
	if _, exist := r.produceTransportIds[userId]; !exist {
		r.produceTransportIds[userId] = transportId
		logger.Infof("create producer transport id=%s for user=%s",
			transportId, userId)
	}
}

func (r *Router) getProducerById(userId string) []*Producer {
	logger.Infof("producers size=%d", len(r.producers[userId]))
	return r.producers[userId]
}

func (r *Router) getConsumerById(userId string) []*Consumer {
	logger.Infof("consumers size=%d", len(r.consumers[userId]))
	return r.consumers[userId]
}

func (r *Router) getOtherConsumersById(userId string) []*Consumer {
	consumers := make([]*Consumer, 0)
	for k, v := range r.consumers {
		if k == userId {
			continue
		}
		for _, consumer := range v {
			consumers = append(consumers, consumer)
		}
	}

	return consumers
}

func (r *Router) getOtherProducersById(userId string) []*Producer {
	producers := make([]*Producer, 0)
	for k, v := range r.producers {
		if k == userId {
			continue
		}
		for _, producer := range v {
			producers = append(producers, producer)
		}
	}

	return producers
}

func (r *Router) getConsumerStreamMid(userId string) string {
	size := len(r.consumers[userId])

	return strconv.Itoa(size)
}

type WorkerStatus int32

const (
	WorkerStatusOffline WorkerStatus = 0
	WorkerStatusOnline  WorkerStatus = 1
	WorkerStatusBusy    WorkerStatus = 2
)

type Worker struct {
	workerId      string
	publicIp      string
	publicPort    uint32
	useUdp        bool
	status        WorkerStatus
	routerCount   uint32
	cpuUsage      uint32
	memoryUsage   uint32
	lastHeartbeat time.Time
	stream        interface{}
	routers       map[string]*Router
	mu            sync.RWMutex
}
