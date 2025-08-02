package main

import (
	"io/ioutil"
	"log"
)

const PDUCount = 100

// func main() {
// 	msgHeader := "\x00\x00\x02\x00\x00\x00\x00\x00"

// 	var input []byte
// 	input = append(input, msgHeader...)

// 	pdu := []byte("\x00\x00\x01\x8A\xA7\xA9\xE9\x48" + "\x00\x00\x01\x58" + "\x00" + "\x00" + "\x00\x08" + "\x00\x00\x00\x00\x80\xFF\x7F\x78")
// 	for i := 0; i < PDUCount; i++ {
// 		input = append(input, pdu...)
// 	}

// 	err := ioutil.WriteFile("/dev/sdpe-eth1", input, 0666)
// 	if err != nil {
// 		log.Println("File open failed !", err)
// 		return
// 	}
// }

func main() {
	msgHeader := "\x00\x00\x02\x00\x00\x00\x00\x00"

	var input []byte
	input = append(input, msgHeader...)

	pdu := []byte("\x00\x00\x01\x8A\xA7\xA9\xE9\x48" + "\x00\x00\x01\x58" + "\x00" + "\x00" + "\x00\x08" + "\x00\x00\x00\x00\x80\xFF\x7F\x78")
	input = append(input, pdu...)
	input = append(input, '\n')

	var result []byte
	for i := 0; i < PDUCount; i++ {
		result = append(result, input...)
	}

	err := ioutil.WriteFile("/dev/sdpe-eth1", result, 0o666)
	if err != nil {
		log.Println("File open failed !", err)
		return
	}
}
