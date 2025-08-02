package can

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"

	"CANAdapter/base"
	"CANAdapter/dbc"
	"CANAdapter/whitelist"

	jsoniter "github.com/json-iterator/go"
	// gojson "github.com/goccy/go-json"
)

var log = base.Logger

type PDU struct {
	UdpTimeStamp uint64
	Timestamp    int64
	CanId        uint32
	BusId        uint8
	Direction    uint8
	PayloadLen   uint16
	Payload      []byte
}

type PDUSlice []PDU

func (x PDUSlice) Len() int           { return len(x) }
func (x PDUSlice) Less(i, j int) bool { return x[i].Timestamp < x[j].Timestamp }
func (x PDUSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

// SDPE Direction
const (
	SDPERecv = iota
	SDPESend
)

// pdu
const (
	TimeStampLen = 8
	CanIdLen     = 4
	BusIdLen     = 1
	DirectionLen = 1
	LengthLen    = 2

	PduHeaderLen = TimeStampLen + CanIdLen + BusIdLen + DirectionLen + LengthLen
)

// byte order
const (
	Motorola = iota
	Intel
)

const MicroPerMilli = 1000

type signal struct {
	signalName  string
	signalValue float64
}

type canFrame struct {
	timeStamp int64
	canName   string
	canId     uint32
	busId     uint8
	direction uint8
	signals   []*signal
	payLoad   []byte
}

var TotalSFrame_3, TotalSFrame_4, TotalSFrame_5, TotalSFrame_6, TotalSFrame_7, TotalSFrame_8 atomic.Int64

func ParseToJson(pdus []PDU) (whiteListJson []byte, otherJson []byte) {
	var otherFrames, whiteListFrames []*canFrame

	for _, pdu := range pdus {
		var frame canFrame
		frame.timeStamp = (pdu.Timestamp / 1000)
		frame.canId = pdu.CanId
		frame.busId = pdu.BusId
		frame.direction = pdu.Direction
		frame.payLoad = pdu.Payload
		canId := pdu.CanId

		if whitelist.IsEnable() {
			if whitelist.QueryByCanId(uint64(canId)) {
				if !decodeCan(canId, &pdu, &frame) {
					continue
				}

				whiteListFrames = append(whiteListFrames, &frame)
				if 2484111111 == pdu.CanId {
					TotalSFrame_6.Add(1)
				}
			} else {
				otherFrames = append(otherFrames, &frame)
			}
		} else {
			if !decodeCan(canId, &pdu, &frame) {
				continue
			}

			whiteListFrames = append(whiteListFrames, &frame)
		}

		if 2484111111 == pdu.CanId {
			TotalSFrame_3.Add(1)
		}
	}

	whiteListJson = toWhiteListJson(whiteListFrames)
	otherJson = toOtherJson(otherFrames)
	return whiteListJson, otherJson
}

func decodeCan(canId uint32, pdu *PDU, frame *canFrame) bool {
	if boVO, ok := dbc.BoVoMap()[uint64(canId)]; !ok {
		log.Warnf("No dbc data !!! canId(%d)", canId)
		return false
	} else {
		if 2484111111 == pdu.CanId {
			TotalSFrame_4.Add(1)
		}
		frame.canName = boVO.CanName
		decodeSignal(boVO, pdu, frame)
		return true
	}
}

func decodeSignal(boVO *dbc.BoVO, pdu *PDU, frame *canFrame) {
	// 按DBC内信号顺序遍历
	for _, sigName := range boVO.OrderedSignals {
		if whitelist.IsEnable() {
			// 开启白名单
			if whitelist.QueryByCanIdAndSignal(uint64(pdu.CanId), sigName) {
				decodeSigValue(sigName, boVO, pdu, frame)
				if 2484111111 == pdu.CanId {
					TotalSFrame_5.Add(1)
				}
			}
		} else {
			// 关闭白名单
			decodeSigValue(sigName, boVO, pdu, frame)
		}
	}
}

func decodeSigValue(sigName string, boVO *dbc.BoVO, pdu *PDU, frame *canFrame) {
	if sigVo, ok := boVO.SgVoMap[sigName]; ok {
		// in whitelist
		var retVal uint64
		startBit := sigVo.StartBit % 8  // 起始位
		startByte := sigVo.StartBit / 8 // 起始字节

		for i := 0; i < sigVo.BitWidth; i++ {
			if startByte < 0 || startByte >= len(pdu.Payload) {
				continue
			}

			if Intel == sigVo.ByteOrder {
				retVal |= uint64(((pdu.Payload[startByte] >> startBit) & 0x01) << i)
				if startBit >= 7 {
					startBit = 0
					startByte++
				} else {
					startBit++
				}
			} else if Motorola == sigVo.ByteOrder {
				retVal |= uint64(((pdu.Payload[startByte] >> startBit) & 0x01) << (sigVo.BitWidth - i - 1))
				if startBit >= 7 {
					startBit = 0
					startByte--
				} else {
					startBit++
				}
			}
		}

		var s signal
		s.signalName = sigName
		s.signalValue = float64(retVal)*sigVo.Factor + float64(sigVo.Offsets)
		frame.signals = append(frame.signals, &s)
	} else {
		log.Warnf("Decode (%s) failed ! not in dbc signal list", sigName)
	}
}

/*
{
	"ts": 1692179443894,
	"raw": {
		"APA_VDC_SYSMTE": "1690681909000 8 00000174 Rx d 8 00 00 00 AA 0D 00 00 00",
		"APA_VDC_SYSMTE2": "1690681909000 8 00000174 Rx d 8 00 00 00 AA 0D 00 00 00"
	},
	"APA_VDC_SYSMTE": {
		"id": 1343,
		"bus": 12,
		"d": 0,
		"t": 1692179443894,
		"DistToDsttnNav": 0,
		"DsttnTypOfNav": 0
	},
	"APA_VDC_SYSMTE2": {
		"id": 1343,
		"bus": 12,
		"d": 0,
		"t": 1692179443894,
		"ChrgngSttnCapctOfDsttnNav": 0,
		"FICMChrgCtrlReq": 0
	}
}
*/

const (
	OpenBrace    = "{"
	ClosingBrace = "}"
)

type CanData struct {
	CanId     uint32 `json:"id"`
	BusId     uint8  `json:"bus"`
	Direction uint8  `json:"d"`
	TimeStamp int64  `json:"t"`
	Signals   map[string]any
}

type JsonData struct {
	TimeStamp int64             `json:"ts"`
	Raw       map[string]string `json:"raw"`
	Attr      map[string]*CanData
}

func (j *JsonData) MarshalJSON() ([]byte, error) {
	datas := make(map[string]any)
	datas["ts"] = j.TimeStamp
	datas["raw"] = j.Raw

	for k, v := range j.Attr {
		cans := make(map[string]any)
		cans["id"] = v.CanId
		cans["bus"] = v.BusId
		cans["d"] = v.Direction
		cans["t"] = v.TimeStamp
		for k, v := range v.Signals {
			cans[k] = v
		}

		datas[k] = cans
	}

	return json.Marshal(datas)
}

func toWhiteListJson(canFrames []*canFrame) (retJson []byte) {
	if len(canFrames) <= 0 || nil == canFrames {
		return nil
	}

	timeStamp := canFrames[0].timeStamp

	jData := &JsonData{
		TimeStamp: timeStamp,
		Raw:       make(map[string]string),
		Attr:      make(map[string]*CanData),
	}

	for _, frame := range canFrames {
		//***************************raw data*******************************
		//"1690681909000 8 00000174 Rx d 8 00 00 00 AA 0D 00 00 00"
		strTimeStamp := strconv.FormatInt(frame.timeStamp, 10)
		canId := strconv.FormatUint(uint64(frame.canId), 10)
		busId := strconv.FormatUint(uint64(frame.busId), 10)

		var strDirection string
		switch frame.direction {
		case SDPERecv:
			strDirection = "Rx d"
		case SDPESend:
			strDirection = "Tx d"
		default:
		}
		dataLen := strconv.FormatUint(uint64(len(frame.payLoad)), 10)

		rawData := bytes.NewBuffer(make([]byte, 0))
		rawData.Write([]byte(strTimeStamp))
		rawData.Write([]byte(" "))
		rawData.Write([]byte(canId))
		rawData.Write([]byte(" "))
		rawData.Write([]byte(busId))
		rawData.Write([]byte(" "))
		rawData.Write([]byte(strDirection))
		rawData.Write([]byte(" "))
		rawData.Write([]byte(dataLen))
		rawData.Write([]byte(" "))
		for _, oneByte := range frame.payLoad {
			rawData.Write(byteToHexChar(oneByte))
			rawData.Write([]byte(" "))
		}
		// pop the last space
		raw := bytes.TrimSpace(rawData.Bytes())

		//"APA_VDC_SYSMTE": "1690681909000 8 00000174 Rx d 8 00 00 00 AA 0D 00 00 00",
		jData.Raw[frame.canName] = string(raw)

		//***************************can frame*******************************
		canData := CanData{
			frame.canId,
			frame.busId,
			frame.direction,
			timeStamp,
			make(map[string]any),
		}

		for _, signal := range frame.signals {
			canData.Signals[signal.signalName] = signal.signalValue
		}
		jData.Attr[frame.canName] = &canData

		if 2484111111 == frame.canId {
			TotalSFrame_7.Add(1)
		}
	}

	// retJson, err := sonic.Marshal(&jData)
	// if err != nil {
	// 	log.Errorln(err)
	// }

	// retJson, err := gojson.Marshal(&jData)
	// if err != nil {
	// 	log.Errorln(err)
	// }

	retJson, err := jsoniter.Marshal(&jData)
	if err != nil {
		log.Errorln(err)
	}

	for _, frame := range canFrames {
		if 2484111111 == frame.canId {
			TotalSFrame_8.Add(1)
		}
	}

	return
}

func toOtherJson(canFrames []*canFrame) (rawJson []byte) {
	if len(canFrames) <= 0 || nil == canFrames {
		return nil
	}
	timeStamp := canFrames[0].timeStamp
	rawData := bytes.NewBuffer(make([]byte, 0))

	var raw []byte
	for _, frame := range canFrames {
		//***************************raw data*******************************
		//"1690681909000 8 00000174 Rx d 8 00 00 00 AA 0D 00 00 00"
		strTimeStamp := strconv.FormatInt(timeStamp, 10)
		canId := strconv.FormatUint(uint64(frame.canId), 10)
		busId := strconv.FormatUint(uint64(frame.busId), 10)

		var strDirection string
		switch frame.direction {
		case SDPERecv:
			strDirection = "Rx d"
		case SDPESend:
			strDirection = "Tx d"
		default:
		}
		dataLen := strconv.FormatUint(uint64(len(frame.payLoad)), 10)

		rawData.Write([]byte(strTimeStamp))
		rawData.Write([]byte(" "))
		rawData.Write([]byte(canId))
		rawData.Write([]byte(" "))
		rawData.Write([]byte(busId))
		rawData.Write([]byte(" "))
		rawData.Write([]byte(strDirection))
		rawData.Write([]byte(" "))
		rawData.Write([]byte(dataLen))
		rawData.Write([]byte(" "))
		for _, oneByte := range frame.payLoad {
			rawData.Write(byteToHexChar(oneByte))
			rawData.Write([]byte(" "))
		}
		// pop the last space
		raw = bytes.TrimSpace(rawData.Bytes())

		// append LF
		raw = append(raw, '\n')
	}

	return raw
}

func byteToHexChar(oneByte byte) []byte {
	high := strings.ToUpper(strconv.FormatUint(uint64(oneByte>>4), 16))
	low := strings.ToUpper(strconv.FormatUint(uint64(oneByte&0x0F), 16))
	return []byte(high + low)
}
