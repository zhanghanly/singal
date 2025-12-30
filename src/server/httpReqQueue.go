package singal

import (
	"sync"
	"time"
)

type Request struct {
	url      string
	data     interface{}
	tryTimes int
}

type ReqQueue struct {
	queue      []*Request
	httpClient *HttpReqMgr
	mutex      sync.RWMutex
}

var gReqQueue *ReqQueue

func NewReqQueue() {
	gReqQueue = &ReqQueue{}

	gReqQueue.queue = make([]*Request, 0)
	gReqQueue.httpClient = NewHttpReqMgr()

	go gReqQueue.DealTimeoutReq()
}

func (req *ReqQueue) PushRequest(rurl string, ldata interface{}) {
	req.mutex.Lock()
	request := &Request{
		url:  rurl,
		data: ldata,
	}
	req.queue = append(req.queue, request)
	logger.Infof("the timeout http req queue size=%d", len(req.queue))

	req.mutex.Unlock()
}

func (req *ReqQueue) PopRequest() *Request {
	req.mutex.Lock()
	request := &Request{}
	if len(req.queue) > 0 {
		logger.Infof("the timeout http req queue size=%d", len(req.queue))
		request.url = req.queue[0].url
		request.data = req.queue[0].data
		req.queue = append(req.queue[:0], req.queue[1:]...)
	}

	req.mutex.Unlock()

	return request
}

func (req *ReqQueue) ExecuteRequest(url string, data interface{}) {
	ret := req.httpClient.sendPost(url, data)
	if ret != 0 {
		// 请求失败，放入失败队列
		req.PushRequest(url, data)
		logger.Errorf("put url=%s, data=%v in queue", url, data)
	}
}

func (req *ReqQueue) DealTimeoutReq() {
	logger.Info("deal timeout req thread start")
	for true {
		request := req.PopRequest()
		if len(request.url) > 0 {
			ret := req.httpClient.sendPost(request.url, request.data)
			if ret != 0 {
				// 请求失败，放入失败队列
				req.PushRequest(request.url, request.data)
				logger.Errorf("put url=%s, data=%v in queue again", request.url, request.data)
			} else {
				logger.Infof("deal timeout req success, url=%s, data=%v in queue again", request.url, request.data)
			}

		}
		time.Sleep(1 * time.Second)
	}
}
