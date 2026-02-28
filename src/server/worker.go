package singal

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
	producers   []*Producer
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

type Worker struct {
	workerId      string
	publicIp      string
	minPort       uint32
	inUsePort     uint32
	maxPort       uint32
	lastAlive     int64
	preferenceUdp bool
	routers       map[string]*Router
}

func (w *Worker) CreateRouter() *Router {
	router := &Router{
		routerId:            w.CreateRouterId(),
		publicIp:            w.publicIp,
		port:                w.inUsePort + 1,
		preferenceUdp:       w.preferenceUdp,
		sendBufSize:         1024,
		recvBufSize:         1024,
		workerId:            w.workerId,
		producers:           make(map[string][]*Producer),
		consumers:           make(map[string][]*Consumer),
		produceTransportIds: make(map[string]string),
		consumeTransportIds: make(map[string]string),
	}

	return router
}

func (w *Worker) CreateRouterId() string {
	for {
		routerId := RandString(36)
		if _, ok := w.routers[routerId]; ok {
			continue
		}

		return routerId
	}
}

func (w *Worker) AddRouter(router *Router) {
	w.routers[router.routerId] = router
}

func (w *Worker) RemoveRouter(router *Router) {
	delete(w.routers, router.routerId)
}
