package media_center

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type Set struct {
	Redis            Redis
	Node             Node
	RegionList       []RegionList
	HttpServerInfo   HttpServerInfo
	HttpCallBackInfo HttpCallBackInfo
	AuthWhiteList    []AuthWhiteList
}

type Redis struct {
	Host      string `yaml:"host"`
	Password  string `yaml:"password"`
	Timeout   int    `yaml:"timeout"`
	MaxActive int    `yaml:"max_active"`
	MaxIdle   int    `yaml:"max_idle"`
	Db        int
}

type Node struct {
	FirstLevel        int `yaml:"firstlevel"`
	SecondLevel       int `yaml:"secondlevel"`
	PushStreamBound   int `yaml:"pushstreambound"`
	BandwidthBound    int `yaml:"bandwidthbound"`
	MaxPushStreamNums int `yaml:"maxpushstreamnums"`
}

type RegionList struct {
	Region     string `yaml:"region"`
	Domain     string `yaml:"domain"`
	DomainPort int    `yaml:"domainport"`
}

type HttpServerInfo struct {
	Port         int `yaml:"port"`
	ReadTimeOut  int `yaml:"readtimeout"`
	WriteTimeOut int `yaml:"writetimeout"`
	IsKeepAlive  int `yaml:"iskeepalive"`
}

type HttpCallBackInfo struct {
	CallBackUrl string `yaml:"callbackurl"`
	Timeout     int    `yaml:"timeout"`
}

// 鉴权白名单
type AuthWhiteList struct {
	Signature string `yaml:"signature"`
}

var gConfig *Set

func InitSetting() {
	file, err := ioutil.ReadFile("../conf/config.yml")
	if err != nil {
		log.Fatal("fail to read file:", err)
	}
	gConfig = &Set{}
	err = yaml.Unmarshal(file, gConfig)
	if err != nil {
		log.Fatal("fail to yaml unmarshal:", err)
	}
}
