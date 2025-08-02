package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	// stu := &map[string]int{
	// 	"peter": 1,
	// 	"jack":  2,
	// 	"tim":   3,
	// 	"tom":   4,
	// }

	stu := &map[string]int{
		"tom":   4,
		"tim":   3,
		"jack":  2,
		"peter": 1,
	}

	for i := 0; i < 50; i++ {
		res, err := json.Marshal(&stu)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(res))
	}
}
