package singal

type MediaType int32

const (
	VIDEO MediaType = 0
	AUDIO MediaType = 1
)

type Producer struct {
	id          string
	transportId string
	mediaType   MediaType
	ssrc        uint32
}

type Consumer struct {
	id          string
	transportId string
	producers   []*Producer
	mediaType   MediaType
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

	producers []*Producer
	consumers []*Consumer
}

func (r *Router) addProducer(producer *Producer) {
	if producer != nil {
		r.producers = append(r.producers, producer)
	}
}

func (r *Router) addConsumer(consumer *Consumer) {
	if consumer != nil {
		r.consumers = append(r.consumers, consumer)
	}
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
		routerId:      w.CreateRouterId(),
		publicIp:      w.publicIp,
		port:          w.inUsePort + 1,
		preferenceUdp: w.preferenceUdp,
		sendBufSize:   1024,
		recvBufSize:   1024,
		workerId:      w.workerId,
		producers:     make([]*Producer, 2),
		consumers:     make([]*Consumer, 2),
	}

	return router
}

func (w *Worker) CreateRouterId() string {
	for {
		routerId := RandString(12)
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
