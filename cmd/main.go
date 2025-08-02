package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	// for performance test
	"net/http"
	_ "net/http/pprof"

	"CANAdapter/base"
	"CANAdapter/can"
	"CANAdapter/dbc"
	"CANAdapter/rwmap"
	"CANAdapter/whitelist"

	"github.com/eclipse/paho.golang/packets"
	"github.com/eclipse/paho.golang/paho"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

var DbcContent []byte

const (
	// udp recv buf
	BufSize    = 8 * 1024
	HeaderLen  = 8
	ConfigPath = "./config.json"
)

// msg type
const (
	ETHSendFrame = iota + 1
	CanMirrorToETH
)

type PDUMapType map[uint32]can.PDU

type RecvData struct {
	RecvTime int64
	Data     []byte
}

var (
	CanDataChan   = make(chan RecvData, base.GConfig.DataChanSize)
	MergedPDUChan = make(chan []can.PDU, base.GConfig.DataChanSize)
	signals       = make(chan os.Signal, 1)
	done          = make(chan struct{})
)

var (
	wg  sync.WaitGroup
	log = base.Logger
)

var TotalPdus atomic.Int64

var totalUdp int
var totalLoseUdp, totalRecvUdpInDecode, totalFrame, totalLoseCAN, TotalSFrame_1, totalMergeRecv, totalMergeSRecv, totalMerge, totalLoseMerge, TotalSFrame_2 atomic.Int64
var totalDLlessHL, totalMsgTypeErr, totalPDLlessPDHL, totalCAN1728, totalPLLlessPLL, totalPDLlessPDL atomic.Int64
var totalMapAssign atomic.Int64

func init() {
	log.SetReportCaller(true)

	switch base.GConfig.Format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: base.TimestampFormat,
		})
	case "text":
		fallthrough
	default:
		log.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: base.TimestampFormat,
		})
	}
}

func main() {
	// debug.SetGCPercent(50)
	if !loadConfig() {
		return
	}
	fmt.Println("Load config success !!!")

	logFile, err := initLog()
	defer logFile.Close()
	if err != nil {
		os.Exit(1)
	}
	log.Debugln("Init log success !!!")

	if ok, err := whitelist.Init(base.GConfig.WhiteListFile, &wg, base.GConfig.EnableWhiteList); !ok {
		log.Errorln(err)
		return
	}

	if base.GConfig.TestMode {
		startPProf(&base.GConfig.PProf)
	}

	// Trap SIGINT to trigger a graceful shutdown.
	signal.Notify(signals, os.Interrupt)
	wg.Add(1)
	go handleQuit(&wg)

	bRet := false
	if base.GConfig.EmbedDBC {
		bRet = loadEmbedDBC()
	} else {
		bRet = loadDBC()
	}

	if !bRet {
		log.Debugln("Load DBC failed !!!")
		os.Exit(1)
	}
	log.Debugln("Load DBC success !!!")

	client := initMQTT()
	defer client.Disconnect(&paho.Disconnect{ReasonCode: 0})

	wg.Add(1)
	go startHttpServer(&wg)

	if base.GConfig.CalcFrameRate {
		ticker := calcFrameRate(base.GConfig.CalcFrameRateInterval, &TotalPdus)
		defer ticker.Stop()
	}

	// handle udp frame
	for i := 0; i < base.GConfig.WorkRoutines; i++ {
		wg.Add(1)
		go handleData(client)
	}

	go readData()

	wg.Wait()
}

func calcFrameRate(interval int, totalPdus *atomic.Int64) *time.Ticker {
	t := time.NewTicker(time.Duration(interval) * time.Second)

	go func(t *time.Ticker) {
		for {
			<-t.C
			log.Errorf("(%f)fps", float64(totalPdus.Load())/float64(interval))
			totalPdus.Store(0)
		}
	}(t)

	return t
}

type PDUPool struct {
	pool *sync.Pool
}

func NewPDUPool(cap int) *PDUPool {
	return &PDUPool{
		pool: &sync.Pool{
			New: func() any {
				buffer := make([]can.PDU, 0, cap)
				return &buffer
			},
		},
	}
}

func (p *PDUPool) Get() *[]can.PDU {
	return p.pool.Get().(*[]can.PDU)
}

func (p *PDUPool) Put(obj *[]can.PDU) {
	p.pool.Put(obj)
}

func readData() {
	sdpeHandle, udpHandle, err := initInterface(base.GConfig.SdpeMode)
	if err != nil {
		log.Fatalln(err)
	}

	if base.GConfig.SdpeMode {
		if sdpeHandle != nil {
			defer sdpeHandle.Close()
		}
	} else {
		if udpHandle != nil {
			defer udpHandle.Close()
		}
	}

	SpecialCANMap := rwmap.NewRWMap(128)
	for _, canId := range base.GConfig.SpecialCANs {
		SpecialCANMap.Set(int64(canId), true)
	}

	pduPool := NewPDUPool(32)
	PDUChan := make(chan *[]can.PDU, base.GConfig.DataChanSize)
	for i := 0; i < base.GConfig.DecodeUdpRoutines; i++ {
		go DecodeUdpData(CanDataChan, PDUChan, SpecialCANMap, pduPool)
	}

	var oldest, reset int64
	pduMap := make(PDUMapType, 2*1024)
	go MergeFrame(PDUChan, &pduMap, pduPool, &oldest, &reset)

	var n int
	var addr net.Addr
	var readErrCnt int

	buf := make([]byte, 2*1024)
	for {
		if base.GConfig.SdpeMode {
			n, err = sdpeHandle.Read(buf)
		} else {
			n, addr, err = udpHandle.ReadFrom(buf)
		}

		log.Debugf("Read %d byte data, err(%v)", n, err)
		var recvData RecvData
		recvData.RecvTime = time.Now().UnixMicro()

		totalUdp++
		if err != nil {
			if io.EOF == err {
				log.Debugln(err, addr.Network(), " ", addr.String())
				continue
			}
			readErrCnt++
			sdpeHandle, udpHandle, _ = initInterface(base.GConfig.SdpeMode)

			//防止重试打开失败后,日志撑爆硬盘
			if readErrCnt <= 10 {
				log.Errorln(err, addr.Network(), " ", addr.String())
			}
		}
		readErrCnt = 0

		if n <= 0 {
			continue
		}

		recvData.Data = make([]byte, n)
		copy(recvData.Data, buf)

		select {
		case CanDataChan <- recvData:
		default:
			totalLoseUdp.Add(1)
		}
	}
}

func initInterface(isSDPE bool) (sdpeHandle *os.File, udpHandle net.PacketConn, err error) {
	if isSDPE {
		// init sdpe
		sdpeHandle, err = os.Open(base.GConfig.SdpeEth)
		if err != nil {
			return
		}
		log.Debugf("Open %s success !!!", base.GConfig.SdpeEth)
	} else {
		// init udp server
		cfg := net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				return c.Control(func(fd uintptr) {
					syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEADDR, 1)
					syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
				})
			},
		}

		udpHandle, err = cfg.ListenPacket(context.Background(), "udp", base.GConfig.UdpServer.Host)
		if err != nil {
			return
		}

		log.Debugf("Open %s success !!!", base.GConfig.UdpServer.Host)
	}

	return
}

func handleData(client *paho.Client) {
	for mergedPdus := range MergedPDUChan {
		parseAndPublish(mergedPdus, client)
	}

	log.Debugln("HandleData quit !!!")
}

func DecodeUdpData(dataChan <-chan RecvData, outChan chan<- *[]can.PDU, specialCANMap *rwmap.RWMap, pduPool *PDUPool) {
	for data := range dataChan {
		if len(data.Data) > 0 {
			totalRecvUdpInDecode.Add(1)
		}

		if len(data.Data) <= 0 {
			continue
		}

		allPdu := decodeUdpData(data.Data, data.RecvTime, specialCANMap, pduPool)
		if len(*allPdu) <= 0 {
			pduPool.Put(allPdu)
			continue
		}

		select {
		case outChan <- allPdu:
		default:
			totalLoseCAN.Add(int64(len(*allPdu)))
		}
	}
}

func decodeUdpData(data []byte, timeStamp int64, specialCANMap *rwmap.RWMap, pduPool *PDUPool) *[]can.PDU {
	if len(data) <= HeaderLen {
		log.Errorf("Invalid data !!! dataLen(%d)", len(data))
		totalDLlessHL.Add(1)
		return nil
	}

	allPdu := pduPool.Get()
	*allPdu = (*allPdu)[:0]

	msgType := data[2]
	// recv can data
	if CanMirrorToETH != msgType {
		log.Errorf("Unknown msg type !!! msgType(%d)", msgType)
		totalMsgTypeErr.Add(1)
		return nil
	}

	pdus := data[HeaderLen:]

	for len(pdus) > 0 {
		if len(pdus) < can.PduHeaderLen {
			log.Errorf("Invalid data !!! dataLen(%d)", len(pdus))
			totalPDLlessPDHL.Add(1)
			break
		}

		var pdu can.PDU
		pdu.Timestamp = timeStamp
		pdu.UdpTimeStamp = binary.BigEndian.Uint64(pdus[:can.TimeStampLen])
		pdu.CanId = binary.BigEndian.Uint32(pdus[can.TimeStampLen : can.TimeStampLen+can.CanIdLen])
		pdu.BusId = pdus[can.TimeStampLen+can.CanIdLen]
		pdu.Direction = pdus[can.PduHeaderLen-can.LengthLen-can.DirectionLen]
		pdu.PayloadLen = binary.BigEndian.Uint16(pdus[can.PduHeaderLen-can.LengthLen : can.PduHeaderLen])
		pudLen := can.PduHeaderLen + int(pdu.PayloadLen)

		if len(pdus) < pudLen {
			log.Errorf("Invalid data !!! pudLen want(%d), has(%d), canId(%d)", pudLen, len(pdus), pdu.CanId)
			totalPDLlessPDL.Add(1)
			break
		}
		pdu.Payload = pdus[can.PduHeaderLen:pudLen]

		if len(pdu.Payload) != int(pdu.PayloadLen) {
			log.Errorf("Invalid data !!! PayloadLen want(%d), has(%d), canId(%d)", int(pdu.PayloadLen), len(pdu.Payload), pdu.CanId)
			totalPLLlessPLL.Add(1)
			break
		}

		pdus = pdus[pudLen:]

		TotalPdus.Add(1)
		totalFrame.Add(1)
		if 2484111111 == pdu.CanId {
			TotalSFrame_1.Add(1)
		}

		if base.GConfig.Bidirection {
			*allPdu = append(*allPdu, pdu)
		} else {
			switch pdu.Direction {
			case can.SDPERecv:
				*allPdu = append(*allPdu, pdu)
			case can.SDPESend:
				if _, ok := specialCANMap.Get(int64(pdu.CanId)); ok {
					*allPdu = append(*allPdu, pdu)
				}
			default:
				log.Errorf("Unknown direction !!! canId(%d), direction(%d)", pdu.CanId, pdu.Direction)
			}
		}

		log.Debugf("TotalRecv(%d)!!!  recvCanId(%d), pduLength(%d), PayloadLen(%d), Payload:%v", totalFrame.Load(), pdu.CanId, pudLen, len(pdu.Payload), pdu.Payload)
	}

	return allPdu
}

func MergeFrame(dataChan <-chan *[]can.PDU, pduMap *PDUMapType, pduPool *PDUPool, oldest, reset *int64) {
	for data := range dataChan {
		totalMergeRecv.Add(int64(len(*data)))
		mergedPdus := mergeFrame(data, pduMap, pduPool, oldest, reset)
		if len(mergedPdus) <= 0 {
			continue
		}

		select {
		case MergedPDUChan <- mergedPdus:
		default:
			totalLoseMerge.Add(int64(len(mergedPdus)))
		}
	}
}

func mergeFrame(inPdus *[]can.PDU, pduMap *PDUMapType, pduPool *PDUPool, oldest, reset *int64) (outPdus []can.PDU) {
	defer pduPool.Put(inPdus)

	latest := (*inPdus)[0].Timestamp
	log.Debugf("PduMap's len(%d), latest(%d), oldest(%d)", len(*pduMap), latest, *oldest)
	if (latest - *oldest) >= int64(base.GConfig.FilterInterval*1000) {
		for _, v := range *pduMap {
			if 2484111111 == v.CanId {
				TotalSFrame_2.Add(1)
			}
			totalMerge.Add(1)
			outPdus = append(outPdus, v)
		}

		log.Debugf("Filtered pdu count(%d), latest(%d), oldest(%d)", len(outPdus), latest, *oldest)
		// 定期重置map,防止map过大
		if (latest - *reset) >= int64(base.GConfig.ResetMapInterval) {
			*pduMap = PDUMapType{}
			*reset = latest
		} else {
			for k := range *pduMap {
				delete(*pduMap, k)
			}
		}
		*oldest = latest
	}

	// deduplicate by canid
	for _, pdu := range *inPdus {
		if 2484111111 == pdu.CanId {
			totalMergeSRecv.Add(1)
		}
		totalMapAssign.Add(1)
		(*pduMap)[pdu.CanId] = pdu
	}

	return
}

func parseAndPublish(mergedPdus []can.PDU, client *paho.Client) {
	whiteListData, otherData := can.ParseToJson(mergedPdus)
	if len(whiteListData) > 0 {
		if _, err := client.Publish(context.Background(), &paho.Publish{
			Topic:   base.GConfig.WhiteList.Topic,
			QoS:     byte(base.GConfig.WhiteList.Qos),
			Retain:  base.GConfig.WhiteList.Retained,
			Payload: whiteListData,
		}); err != nil {
			log.Errorln("WhiteList error sending message: ", err)
		}

		//********************************************
		// go func(msg []byte, client *paho.Client) {
		// 	if _, err := client.Publish(context.Background(), &paho.Publish{
		// 		Topic:   base.GConfig.WhiteList.Topic,
		// 		QoS:     byte(base.GConfig.WhiteList.Qos),
		// 		Retain:  base.GConfig.WhiteList.Retained,
		// 		Payload: msg,
		// 	}); err != nil {
		// 		log.Errorln("WhiteList error sending message: ", err)
		// 	}
		// }(whiteListData, client)
	} else {
		log.Debugln("No white List data")
	}

	if len(otherData) > 0 {
		if _, err := client.Publish(context.Background(), &paho.Publish{
			Topic:   base.GConfig.NonWhiteList.Topic,
			QoS:     byte(base.GConfig.NonWhiteList.Qos),
			Retain:  base.GConfig.NonWhiteList.Retained,
			Payload: otherData,
		}); err != nil {
			log.Errorln("NonWhiteList error sending message: ", err)
		}

		//********************************************
		// go func(msg []byte, client *paho.Client) {
		// 	if _, err := client.Publish(context.Background(), &paho.Publish{
		// 		Topic:   base.GConfig.NonWhiteList.Topic,
		// 		QoS:     byte(base.GConfig.NonWhiteList.Qos),
		// 		Retain:  base.GConfig.NonWhiteList.Retained,
		// 		Payload: msg,
		// 	}); err != nil {
		// 		log.Errorln("NonWhiteList error sending message: ", err)
		// 	}
		// }(otherData, client)
	} else {
		log.Debugln("No raw data")
	}
}

func handleQuit(wg *sync.WaitGroup) {
	defer wg.Done()

	select {
	case <-signals:
		log.Debugln("recv interrupt signal")
		log.Errorf(`totalUdp(%d), totalLoseUdp(%d), totalRecvUdpInDecode(%d), totalFrame(%d), TotalSFrame_1(%d), totalLoseCAN(%d), totalMergeRecv(%d), totalMergeSRecv(%d), 
		totalMerge(%d), totalLoseMerge(%d), TotalSFrame_2(%d), TotalSFrame_3(%d), TotalSFrame_4(%d), TotalSFrame_5(%d), TotalSFrame_6(%d), TotalSFrame_7(%d), TotalSFrame_8(%d) || 
		totalDLlessHL(%d), totalMsgTypeErr(%d), totalPDLlessPDHL(%d), totalCAN1728(%d), totalPDLlessPDL(%d), totalPLLlessPLL(%d), totalMapAssign(%d)`,
			totalUdp, totalLoseUdp.Load(), totalRecvUdpInDecode.Load(),
			totalFrame.Load(), TotalSFrame_1.Load(), totalLoseCAN.Load(), totalMergeRecv.Load(), totalMergeSRecv.Load(), totalMerge.Load(), totalLoseMerge.Load(), TotalSFrame_2.Load(),
			can.TotalSFrame_3.Load(), can.TotalSFrame_4.Load(), can.TotalSFrame_5.Load(), can.TotalSFrame_6.Load(),
			can.TotalSFrame_7.Load(), can.TotalSFrame_8.Load(),
			totalDLlessHL.Load(),
			totalMsgTypeErr.Load(),
			totalPDLlessPDHL.Load(),
			totalCAN1728.Load(),
			totalPDLlessPDL.Load(),
			totalPLLlessPLL.Load(),
			totalMapAssign.Load(),
		)

		time.Sleep(5 * time.Second)
		close(done)
		os.Exit(0)
	}
}

func loadConfig() bool {
	jData, err := ioutil.ReadFile(ConfigPath)
	if err != nil {
		fmt.Println(err)
		return false
	}

	err = json.Unmarshal(jData, base.GConfig)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func loadDBC() bool {
	f, err := os.Open(base.GConfig.DBCPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	return dbc.NewParser(f).Parse()
}

func loadEmbedDBC() bool {
	if len(DbcContent) <= 0 {
		return false
	}

	r := bytes.NewReader(DbcContent)
	return dbc.NewParser(r).Parse()
}

func loadDBCByExcel() bool {
	return dbc.ParseExcel(base.GConfig.DBCExcel)
}

func initLog() (io.ReadWriteCloser, error) {
	if len(os.Args) < 1 {
		return nil, errors.New("Invalid Args")
	}

	var err error
	var logFile *os.File
	if base.GConfig.LogToFile {
		err = os.MkdirAll("./log", os.ModePerm)
		if err != nil {
			return nil, err
		}

		logName := "./log/" + filepath.Base(os.Args[0])
		now := time.Now()
		strTime := time.Unix(now.Unix(), now.UnixNano()).Format(base.TimestampFormat)
		strTime = strings.Replace(strTime, ":", "_", -1)
		logName += "." + strTime + ".log"

		logFile, err = os.OpenFile(logName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o666)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Open %s success !\n", logName)

		log.SetOutput(logFile)
		log.Printf("Open %s success !\n", logName)
	}

	if level, err := logrus.ParseLevel(base.GConfig.LogLevel); err != nil {
		fmt.Println("ParseLevel failed !!! ", base.GConfig.LogLevel, err)
		return logFile, err
	} else {
		log.SetLevel(level)
	}

	return logFile, nil
}

func startPProf(pprof *base.PProf) {
	server := &HttpServer{
		Server: &http.Server{
			Addr:    pprof.Addr,
			Handler: nil,
		},
	}

	go server.WaitExitSignal(pprof.Timeout * time.Second)
	go func(server *HttpServer) {
		err := server.ListenAndServe()
		if err != nil {
			log.Errorln("unexpected error from ListenAndServe: ", "reason:", err)
		}

		log.Debugln("main goroutine exited.")
	}(server)
}

func initMQTT() *paho.Client {
	tcpConn, err := net.Dial("tcp", base.GConfig.Broker)
	if err != nil {
		log.Fatalln("Failed to connect to ", base.GConfig.Broker, "reason:", err)
	}
	log.Debugln("Success to connect to ", base.GConfig.Broker)

	tcpConn = packets.NewThreadSafeConn(tcpConn)

	client := paho.NewClient(paho.ClientConfig{
		Conn: tcpConn,
	})

	cp := &paho.Connect{
		KeepAlive:  30,
		ClientID:   base.GConfig.Clientid,
		CleanStart: true,
		Username:   base.GConfig.Username,
		Password:   []byte(base.GConfig.Password),
	}

	if base.GConfig.Username != "" {
		cp.UsernameFlag = true
	}
	if base.GConfig.Password != "" {
		cp.PasswordFlag = true
	}

	log.Debugln("UsernameFlag:", cp.UsernameFlag, " PasswordFlag:", cp.PasswordFlag)

	ca, err := client.Connect(context.Background(), cp)
	if err != nil {
		log.Fatalln(err)
	}

	if ca.ReasonCode != 0 {
		log.Fatalf("Failed to connect to %s : %d - %s", base.GConfig.Broker, ca.ReasonCode, ca.Properties.ReasonString)
	}

	log.Debugf("Connected to %s\n", base.GConfig.Broker)
	return client
}

func startHttpServer(wg *sync.WaitGroup) {
	defer wg.Done()
	http.HandleFunc(base.GConfig.HttpServer.HealthCheckURI, Pong)                 // ping路由
	http.HandleFunc(base.GConfig.HttpServer.WhiteListURI, whitelist.SetWhiteList) // 设置白名单
	http.ListenAndServe(base.GConfig.HttpServer.ServerAddr, nil)
}

func Pong(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
