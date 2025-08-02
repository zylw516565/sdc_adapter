package dbc

import (
	"strconv"

	"github.com/xuri/excelize/v2"
)

const (
	CanId = iota
	CanName
	PeriodOfTx
	MsgLen
	StartByte
	StartBit
	BitWidth
	SignalName
	SignalSymbol
	TransmitterECU
	ExcelMaxColumn
)

func init() {
	if dbcData.BoVoMap == nil {
		dbcData.BoVoMap = make(map[uint64]*BoVO)
	}
}

func ParseExcel(filename string) bool {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		log.Errorln(err)
		return false
	}
	defer f.Close()

	//获取DBC Sheet上所有单元格
	rows, err := f.GetRows("DBC")
	if err != nil {
		log.Errorln(err)
		return false
	}

	for idx, row := range rows {
		if idx <= 0 {
			continue
		}

		if len(row) < ExcelMaxColumn {
			log.Errorf("Invalid number of columns! want(%d), has(%d)", ExcelMaxColumn, len(row))
			return false
		}

		var boVO BoVO
		boVO.CanId, _ = strconv.ParseUint(row[CanId], 10, 64)
		boVO.CanName = row[CanName]
		boVO.DataLenth, _ = strconv.ParseUint(row[MsgLen], 10, 64)

		var sgVO SgVO
		sgVO.StartBit, _ = strconv.Atoi(row[StartBit])
		sgVO.BitWidth, _ = strconv.Atoi(row[BitWidth])
		sgVO.SignalName = row[SignalName]
		// sgVO.ByteOrder,
		// sgVO.ValueType
		// sgVO.Factor, _
		// sgVO.Offsets, _
		// sgVO.Min, _ = s
		// sgVO.Max, _ = s
		// sgVO.Unit = str
		// sgVO.Receiver =
		// sgVO.DbcContent

		if _, ok := dbcData.BoVoMap[boVO.CanId]; !ok {
			dbcData.BoVoMap[boVO.CanId] = &boVO
		} else {
			dbcData.BoVoMap[boVO.CanId].SgVoMap[sgVO.SignalName] = &sgVO
		}
	}

	return true
}
