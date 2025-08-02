package base

import (
	"time"
)

type MQTTTopic struct {
	Topic    string
	Qos      int
	Retained bool
}

type MQTT struct {
	WhiteList    MQTTTopic
	NonWhiteList MQTTTopic
	Broker       string
	Clientid     string
	Username     string
	Password     string
}

type HttpServer struct {
	ServerAddr     string // in the form "host:port"
	Method         string
	HealthCheckURI string // default: ping
	WhiteListURI   string
}

type LOG struct {
	LogToFile bool
	Format    string // json, text
	LogLevel  string // panic, fatal, error, warn warning, info, debug, trace
}

type PProf struct {
	Addr    string
	Timeout time.Duration
}

type TEST struct {
	TestMode        bool
	EnableWhiteList bool
	PProf           `json:"PProf"`
}

type UdpServer struct {
	Host          string
	NumLoops      int
	IdleTime      time.Duration
	MetricsServer string
}

type DataSoure struct {
	SdpeMode  bool
	SdpeEth   string
	UdpServer `json:"UdpServer"`
}

type Filter struct {
	IsFilterFrame    bool
	FilterInterval   int
	ResetMapInterval int
}

type DBC struct {
	EmbedDBC bool //true:把dbc文件内嵌到go程序, false:不内嵌dbc文件
	DBCPath  string
	DBCExcel string
}

type Config struct {
	MQTT                  `json:"MQTT"`
	HttpServer            `json:"HttpServer"`
	DataChanSize          uint
	WorkRoutines          int
	DecodeUdpRoutines     int
	DBC                   `json:"DBC"`
	WhiteListFile         string
	LOG                   `json:"LOG"`
	Filter                `json:"Filter"`
	Bidirection           bool
	CalcFrameRate         bool
	CalcFrameRateInterval int
	TEST                  `json:"TEST"`
	DataSoure             `json:"DataSoure"`
	SpecialCANs           []int
}

func NewConfig() *Config {
	return &Config{
		MQTT{},
		HttpServer{},
		10000,
		10,
		10,
		DBC{true, "./can.dbc", "./can.xlsx"},
		"./whitelist.json",
		LOG{false, "text", "info"},
		Filter{true, 10, 600000},
		false,
		true,
		5,
		TEST{},
		DataSoure{},
		[]int{},
	}
}

var GConfig = NewConfig()
