package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
)

const Count = 78

var Sig string = ` EAT_158: 8 VCU
 SG_ EAT_CHECKSUM_158 : 59|4@0+ (1,0) [0|15] ""  OBC
 SG_ EAT_ALIVE_COUNTER_158 : 61|2@0+ (1,0) [0|3] ""  OBC
 SG_ EAT_TRANS_SPEED : 39|16@0+ (0.01,0) [0|655.35] ""  OBC

`

var DDC string = `: 8 DCDC
 SG_ DDC12V_TDV : 7|8@0+ (1,-50) [-50|205] "℃"  VCU
 SG_ DDC12V_IIN : 15|8@0+ (0.1,0) [0|25.5] "A"  VCU
 SG_ DDC12V_IOUT : 23|8@0+ (1,0) [0|255] "A"  BMS,VCU
 SG_ DDC12V_CHG_LAMP : 30|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_VO_LV : 29|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_DV_ERR : 28|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_OT_1 : 27|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_OC : 26|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_TS_ERR : 25|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_RX_ERR : 24|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_CAN_NON : 39|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_DV_INH : 38|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_IG_LV : 37|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_IG_UV : 36|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_OT_2 : 35|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_VO_OV : 34|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_VI_OV : 33|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_VI_LV : 32|1@0+ (1,0) [0|1] ""  VCU
 SG_ DDC12V_CAN_ERR : 47|8@0+ (1,0) [0|255] ""  VCU
 SG_ DDC12V_DVC_OUT : 55|8@0+ (0.1,0) [0|25.5] "V"  VCU
 SG_ DDC12V_ALIVE_COUNTER_22E : 61|2@0+ (1,0) [0|3] ""  BMS,VCU
 SG_ DDC12V_CHECKSUM_22E : 59|4@0+ (1,0) [0|15] ""  BMS,VCU

`

// BO_ 558 DDC12V_22E
func main() {
	var canId uint32 = 2484000000
	fmt.Println(canId)
	canNamePrefix := "DDC12V_"

	Data := []byte{}

	for i := 0; i < Count; i++ {
		strCanId := strconv.FormatUint(uint64(canId), 10)
		fullSig := "BO_ " + strCanId + " " + canNamePrefix + strCanId + DDC
		Data = append(Data, []byte(fullSig)...)
		canId--
	}

	err := ioutil.WriteFile("./data.txt", Data, 0o666)
	if err != nil {
		log.Println("File open failed !", err)
		return
	}
}
