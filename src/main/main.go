package main

import (
	"runtime"
	singal "singal/src/server"
	"sync"
)

var (
	ProjectName  string //应用名称
	BuildVersion string //编译版本
	GitBranch    string //Git分支
	BuildTime    string //编译时间
	GoVersion    string //Golang信息
)

func panicIfError(err error) {
	if err != nil {
		singal.GetLogger().Error("error: %v", err)
		panic(err)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	singal.InitLogger()

	var logger = singal.GetLogger()
	logger.Infof("Project Name: %s", ProjectName)
	logger.Infof("Build version: %s", BuildVersion)
	logger.Infof("Git branch: %s", GitBranch)
	logger.Infof("Build time: %s", BuildTime)
	logger.Infof("Golang Version: %s", GoVersion)
	logger.Info("starting httpserver.")

	if err := singal.InitSetting(); err != nil {
		logger.Errorf("failed to initialize setting: %v", err)
		return
	}
	//if err := singal.InitRedisClient(); err != nil {
	//	logger.Errorf("failed to initialize redis: %v", err)
	//	return
	//}
	//if err := singal.InitAuthDB(); err != nil {
	//	logger.Errorf("failed to initialize auth DB: %v", err)
	//	return
	//}

	singal.NewRoomManager()
	//singal.InitEmailService()

	var wg sync.WaitGroup
	wg.Add(3)
	// start WebSocket server
	go func() {
		defer wg.Done()
		singal.StartWssServer()
	}()
	// start grpc server
	go func() {
		defer wg.Done()
		singal.StartGrpcServer()
	}()
	// start http server
	go func() {
		defer wg.Done()
		singal.StartHttpServer()
	}()

	wg.Wait()
}
