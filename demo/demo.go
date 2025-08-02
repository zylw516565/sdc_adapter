package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	// init sdpe
	sdpe, err := os.OpenFile("/dev/sdpe-eth1", os.O_RDONLY, 0o666)
	if err != nil {
		log.Fatalln(err)
	}
	defer sdpe.Close()
	log.Printf("Open %s success !!!", "/dev/sdpe-eth1")

	buf := make([]byte, 1024)
	r := bufio.NewReader(sdpe)

	for {
		n, err := r.Read(buf)
		if err != nil {
			if io.EOF != err {
				log.Println(err)
			}
			continue
		}

		fmt.Println("redcv", buf[:n])
	}
}
