package main

import (
	"runtime"
	singal "singal/src/server"
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
	//初始化日志
	singal.InitLogger()

	var logger = singal.GetLogger()
	//打印版本信息
	logger.Infof("Project Name: %s", ProjectName)
	logger.Infof("Build version: %s", BuildVersion)
	logger.Infof("Git branch: %s", GitBranch)
	logger.Infof("Build time: %s", BuildTime)
	logger.Infof("Golang Version: %s", GoVersion)
	logger.Info("starting httpserver.")

	singal.NewRoomManager()
	//加载配置
	singal.InitSetting()
	//if err := singal.InitRedisClient(); err != nil {
	//	return
	//}
	singal.StartWssServer()
}
