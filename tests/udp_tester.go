package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

var InputData = []byte{
	// Message Header
	0x02, 0x04, // RPC ID
	0x02, 0x00, // RPC Type
	0x00, 0x00, 0x00, 0x00, // RPC Cookie
	// Pdu
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Timestamp
	0x00, 0x00, 0x01, 0x58, // Can Id; 344
	0x00,       // Bus Id
	0x00,       // Direction
	0x00, 0x08, // Length
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Payload
	// Pdu
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Timestamp
	0x00, 0x00, 0x06, 0xc0, // Can Id; 1728
	0x00,       // Bus Id
	0x01,       // Direction
	0x00, 0x08, // Length
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Payload
	// Pdu
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Timestamp
	0x00, 0x00, 0x06, 0xc1, // Can Id; 1728
	0x00,       // Bus Id
	0x01,       // Direction
	0x00, 0x08, // Length
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Payload
}

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	addr := &net.UDPAddr{IP: net.ParseIP("localhost"), Port: 7000}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	var totalBytes, totalCount int64
	packetLen := int64(len(InputData))
	start := time.Now().UnixMilli()
quit:
	for {
		select {
		case <-signals:
			log.Println("recv interrupt signal")
			break quit
		default:
			for singleCount := 0; singleCount < 30; singleCount++ {
				fmt.Fprintf(conn, string(InputData))
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
}
