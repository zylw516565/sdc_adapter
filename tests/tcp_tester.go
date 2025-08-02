package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 7000}
	conn, err := net.DialTCP("tcp", nil, addr)
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
			for singleCount := 0; singleCount < 3*13000; singleCount++ {
				fmt.Fprintf(conn, string(input1))
				totalCount++
				totalBytes += packetLen
			}
		}
		time.Sleep(1 * time.Second)
	}

	totalTime := time.Now().UnixMilli() - start
	totalSeconds := float64(totalTime) / float64(1000)
	rate := float64(totalBytes) / (float64(totalTime) / float64(1000))
	frameRate := float64(totalCount) / (float64(totalTime) / float64(1000))
	fmt.Printf("totalCount(%d), totalBytes(%d), totalSeconds(%fs), bitRate(%f B/S, %f bps), frameRate(%f fps)\n", totalCount, totalBytes, totalSeconds, rate, rate*8, frameRate)
}
