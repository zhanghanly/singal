package media_center

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type HttpReqMgr struct {
	client              *http.Client
	maxIdleConns        int
	maxIdleConnsPerHost int
	maxConnsPerHost     int
}

func NewHttpReqMgr() *HttpReqMgr {
	mytransport := http.DefaultTransport.(*http.Transport).Clone()
	gHttpReqMgr := &HttpReqMgr{}

	//最大空闲连接数，默认值为100
	gHttpReqMgr.maxIdleConns = 100
	//跟单个对端节点的能维持的最大空闲连接数， 默认值为2
	gHttpReqMgr.maxIdleConnsPerHost = 2
	//限制单个对端节点的最大连接数, 默认值为0，表示没有限制
	gHttpReqMgr.maxConnsPerHost = 0

	//跟所有对端节点的能维持的最大空闲连接数, 设置成一个合理的值可以减少连接的释放频率 可以减少time_wait的状态的产生
	mytransport.MaxIdleConns = gHttpReqMgr.maxIdleConns
	//跟单个对端节点的能维持的最大的空闲连接数, 不超过MaxIdleConns的数量
	mytransport.MaxIdleConnsPerHost = gHttpReqMgr.maxIdleConnsPerHost
	//限制单个对端节点的最大连接数
	mytransport.MaxConnsPerHost = gHttpReqMgr.maxConnsPerHost

	gHttpReqMgr.client = &http.Client{
		Timeout:   time.Duration(gConfig.HttpCallBackInfo.Timeout) * time.Second,
		Transport: mytransport,
	}

	return gHttpReqMgr
}

func (mgr *HttpReqMgr) sendPost(url string, data interface{}) int {
	traceId := RandString(16)
	//发送请求
	resCode := 0
	bytesData, err := json.Marshal(data)
	if err != nil {
		logger.Errorf("[sendPost] url:%s, data%v is jsonMarshal error:%v", url, data, err)
		return 0
	}

	HttpReq, err := http.NewRequest("POST", url, bytes.NewReader(bytesData))
	HttpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")
	if err != nil {
		logger.Errorf("[sendPost] url:%s, data%v is NewRequest error:%v", url, data, err)
		return 0
	}
	response, err := mgr.client.Do(HttpReq)
	if err != nil {
		netErr, ok := err.(net.Error)
		if ok && netErr.Timeout() {
			resCode = 1
		} else {
			resCode = 2
		}

		logger.Errorf("[sendPost] Send Error traceId:%s err:%v resCode:%v", traceId, err, resCode)
		return -1
	}
	logger.Infof("[sendPost] Response Success traceId:%s", traceId)

	defer response.Body.Close()
	resBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Errorf("[sendPost] Read Response Body Error traceId:%s httpResCode:%d", traceId, response.StatusCode)
		return -1
	}

	//解析http返回码是否正确
	if response.StatusCode != 200 {
		// 解析应答失败的信息
		resCode = 3
		logger.Errorf("[sendPost] Response StatusCodeError traceId:%s httpResCode:%d resp body: %v", traceId, response.StatusCode, string(resBody))
		return -1
	}

	return 0
}
