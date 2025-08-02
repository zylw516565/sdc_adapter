package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

func main() {
	var host string
	flag.StringVar(&host, "h", "0.0.0.0:5566", "请输入host: ")
	flag.Parse()
	fmt.Println("host: ", host)
	udpServer(host)
}

// 限制goroutine数量
var limitChan = make(chan bool, 1000)

// UDP goroutine 实现并发读取UDP数据
func udpProcess(conn *net.UDPConn) {

	// 最大读取数据大小
	data := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(data)
	if err != nil {
		fmt.Println("failed read udp msg, error: " + err.Error())
	}
	str := string(data[:n])
	fmt.Println("receive from client, data:" + str)
	<-limitChan
}

func udpServer(address string) {
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	conn, err := net.ListenUDP("udp4", udpAddr)
	defer conn.Close()
	if err != nil {
		fmt.Println("read from connect failed, err:" + err.Error())
		os.Exit(1)
	}

	for {
		limitChan <- true
		go udpProcess(conn)
	}
}
