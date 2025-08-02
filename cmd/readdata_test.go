package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"testing"
	"time"

	"CANAdapter/base"

	"github.com/sirupsen/logrus"
)

var gConfig = base.NewConfig()

func init() {
	log.SetReportCaller(true)

	// log.SetFormatter(&logrus.TextFormatter{
	// 	FullTimestamp:   true,
	// 	TimestampFormat: base.TimestampFormat,
	// })

	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: base.TimestampFormat,
	})

	gConfig.LogLevel = "fatal"

	go startUDP()
	gConfig.SdpeMode = false
	if gConfig.SdpeMode {
		gConfig.SdpeEth = "/dev/sdpe-eth1"
	} else {
		gConfig.UdpServer.Host = "127.0.0.1:7000"
	}

	if level, err := logrus.ParseLevel(gConfig.LogLevel); err != nil {
		fmt.Println("ParseLevel failed !!! ", gConfig.LogLevel, err)
		return
	} else {
		log.SetLevel(level)
	}
}

func BenchmarkReadData(b *testing.B) {
	b.StopTimer()

	var r io.Reader
	buf := make([]byte, 2*1024)
	sdpeHandle, udpHandle, _ := initInterface(gConfig.SdpeMode)
	if gConfig.SdpeMode {
		if sdpeHandle != nil {
			defer sdpeHandle.Close()
		}

		r = bufio.NewReader(sdpeHandle)
	} else {
		if udpHandle != nil {
			defer udpHandle.Close()
		}
	}

	var n int
	var err error
	var addr net.Addr
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if gConfig.SdpeMode {
			n, err = r.Read(buf)
		} else {
			n, addr, err = udpHandle.ReadFrom(buf)
		}

		if err != nil {
			if io.EOF != err {
				log.Errorln(err, addr.Network(), " ", addr.String())
			}
			log.Debugln(err, addr.Network(), " ", addr.String())
			continue
		}

		if 0 == n {
			continue
		}
	}
}

func BenchmarkDecodeUdpData(b *testing.B) {
	var TotalPdus atomic.Int64
	go readData(&TotalPdus)

	var data []byte
	var totalPdus atomic.Int64

	for i := 0; i < b.N; i++ {
		data = <-CanDataChan
		allPdu := decodeUdpData(data, &totalPdus)
		if len(allPdu) <= 0 {
			continue
		}
	}
}

func startUDP() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 7000}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	msgHeader := "\x00\x00\x02\x00\x00\x00\x00\x00"

	input1 := []byte(msgHeader + "\x00\x00\x01\x8A\xA7\xA9\xE9\x48" + "\x00\x00\x01\x58" + "\x00" + "\x00" + "\x00\x08" + "\x00\x00\x00\x00\x80\xFF\x7F\x78")

	var totalBytes, totalCount int64
	packetLen := int64(len(input1))
	start := time.Now().UnixMilli()
quit:
	for {
		select {
		case <-signals:
			log.Println("recv interrupt signal")
			break quit
		default:
			for singleCount := 0; singleCount < 30; singleCount++ {
				fmt.Fprintf(conn, string(input1))
				totalCount++
				totalBytes += packetLen
			}
		}
		time.Sleep(1 * time.Millisecond)
	}

	totalTime := time.Now().UnixMilli() - start
	totalSeconds := float64(totalTime) / float64(1000)
	rate := float64(totalBytes) / (float64(totalTime) / float64(1000))
	frameRate := float64(totalCount) / (float64(totalTime) / float64(1000))
	fmt.Printf("totalCount(%d), totalBytes(%d), totalSeconds(%fs), bitRate(%f B/S, %f bps), frameRate(%f fps)\n", totalCount, totalBytes, totalSeconds, rate, rate*8, frameRate)
	os.Exit(0)
}
