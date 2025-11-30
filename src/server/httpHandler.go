package media_center

import (
	// "fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CODE_SUCCESS          = int(0)
	CODE_ERROR            = int(400)
	CODE_INVALID_PARAM    = int(401)
	CODE_FORBIDDEN_PARAM  = int(403)
	CODE_REPEAT_PARAM     = int(404)
	CODE_SERVERBUSY_PARAM = int(501)
)

const (
	nodeMaxBandwidth = uint32(100 * 1024) // 单个流媒体节点的带宽(单位：kb)
)

var errMaps map[int]string

type HttpHandler struct {
}

var gHttpHandler *HttpHandler

func NewHttpHandler(router *gin.Engine) (gHttpHandler *HttpHandler) {
	gHttpHandler = &HttpHandler{}
	// api
	router.POST("/echo", gHttpHandler.EchoTest)

	errMaps = make(map[int]string)
	errMaps[CODE_SUCCESS] = "success"
	errMaps[CODE_INVALID_PARAM] = "invalid param"
	errMaps[CODE_FORBIDDEN_PARAM] = "forbidden param"
	errMaps[CODE_REPEAT_PARAM] = "repeat param"
	errMaps[CODE_SERVERBUSY_PARAM] = "server busy"

	return
}

func (hh *HttpHandler) EchoTest(c *gin.Context) {
	logger.Debugf("[EchoTest] c.Request.Method: %v", c.Request.Method)
	logger.Debugf("[EchoTest] c.Request.ContentType: %v", c.ContentType())

	c.Request.ParseForm()
	logger.Debugf("[EchoTest] c.Request.Form: %v", c.Request.PostForm)

	for k, v := range c.Request.PostForm {
		logger.Debugf("[EchoTest] k:%v\n", k)
		logger.Debugf("[EchoTest] v:%v\n", v)
	}

	logger.Debugf("[EchoTest] c.Request.ContentLength: %v", c.Request.ContentLength)
	data, _ := ioutil.ReadAll(c.Request.Body)

	logger.Debugf("[EchoTest] c.Request.GetBody: %v", string(data))
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "echotest", "traceId": "mwkjt"})
}
