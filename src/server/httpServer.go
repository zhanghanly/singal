package singal

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type HttpServer struct {
	Server  *http.Server
	Handler *HttpHandler
}

func accessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		{
			param := gin.LogFormatterParams{
				Request: c.Request,
				Keys:    c.Keys,
			}

			// Stop timer
			param.TimeStamp = time.Now()
			param.Latency = param.TimeStamp.Sub(start)

			param.ClientIP = c.ClientIP()
			param.Method = c.Request.Method
			param.StatusCode = c.Writer.Status()
			param.BodySize = c.Writer.Size()

			if raw != "" {
				path = path + "?" + raw
			}

			param.Path = path

			logger.Infof("[access] %s - [%s] \"%s %s %s\" %d %s %d %s",
				param.ClientIP,
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.BodySize,
				c.Request.Host)
		}
	}
}

func NewHttpServer() *HttpServer {
	hs := &HttpServer{}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	hs.Handler = NewHttpHandler(router)
	hs.Server = &http.Server{
		Addr:         gConfig.Http.Addr,
		Handler:      router,
		ReadTimeout:  time.Duration(gConfig.Http.ReadTimeOut) * time.Second,
		WriteTimeout: time.Duration(gConfig.Http.WriteTimeOut) * time.Second,
	}
	if gConfig.Http.IsKeepAlive == 1 {
		hs.Server.SetKeepAlivesEnabled(true)
	}

	return hs
}

func (hs *HttpServer) Stop() error {
	// gracefully shutdown
	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("shutting down ...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) // shutdown timeout
	defer cancel()

	if err := hs.Server.Shutdown(ctx); err != nil {
		logger.Errorf("shutdown err: %v", err)
	}

	select {
	case <-ctx.Done():
		logger.Info("shutdown ok ...")
	}
	logger.Info("server exit")
	return nil
}

func StartHttpServer() {
	logger.Info("Start http Server ....")

	hs := NewHttpServer()
	if err := hs.Server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			logger.Info("server shutdown completed")
		} else {
			logger.Errorf("server closed, unexpected err: %v", err)
		}
	}
}
