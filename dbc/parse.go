package dbc

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"CANAdapter/base"
)

var log = base.Logger

const (
	kVersion  = "VERSION"
	kNS       = "NS_"
	kBS       = "BS_"
	kBU       = "BU_"
	kValTable = "VAL_TABLE_"
	kBO       = "BO_"
	kSG       = "SG_"
	kCM       = "CM_"
	kVAL      = "VAL_"
)

type Parser struct {
	r   io.Reader
	buf []string
	err error
}

// dbc data
var dbcData DbcVO

func NewParser(r io.Reader) *Parser {
	return &Parser{
		r: r,
	}
}

func (p *Parser) Parse() bool {
	input := bufio.NewReader(p.r)

	// read file
	for {
		// read a line
		line, err := input.ReadString('\n')
		if err == io.EOF {
			var name string
			if f, ok := p.r.(*os.File); ok {
				name = f.Name()
			} else {
				name = ""
			}
			log.Infoln("read EOF from ", name)
			break
		}

		if err != nil {
			log.Errorln(err)
			p.setErr(err)
			return false
		}
		p.buf = append(p.buf, strings.Trim(line, "\r\n"))
	}

	for idx := 0; idx < len(p.buf); idx++ {
		upperLine := strings.ToUpper(strings.TrimSpace(p.buf[idx]))
		if strings.HasPrefix(upperLine, kVersion) {
			p.parseVersion(&idx, p.buf)
		} else if strings.HasPrefix(upperLine, kNS) {
			p.parseNS(&idx, p.buf)
		} else if strings.HasPrefix(upperLine, kBS) {
			p.parseBS(&idx, p.buf)
		} else if strings.HasPrefix(upperLine, kBU) {
			p.parseBU(&idx, p.buf)
		} else if strings.HasPrefix(upperLine, kValTable) {
			p.parseValTable(&idx, p.buf)
		} else if strings.HasPrefix(upperLine, kBO) {
			p.parseBO(&idx, p.buf)
		} else if strings.HasPrefix(upperLine, kCM) {
			p.parseCM(&idx, p.buf)
		} else if strings.HasPrefix(upperLine, kVAL) {
			p.parseVal(&idx, p.buf)
		} else if len(upperLine) != 0 {
			dbcData.UnParseLines = append(dbcData.UnParseLines, "行号:"+strconv.Itoa(idx+1)+" :"+p.buf[idx])
		}
	}

	return true
}

func (p *Parser) parseVersion(curIdx *int, buf []string) {
	subs := strings.Split(buf[*curIdx], "\"")
	if len(subs) > 1 {
		dbcData.Version = subs[1]
	}
}

func (p *Parser) parseNS(curIdx *int, buf []string) {
	for ; *curIdx < len(buf); (*curIdx)++ {
		line := buf[*curIdx]
		if strings.HasPrefix(strings.ToUpper(line), kNS) {
			continue
		}

		if "" == line {
			return
		}

		dbcData.Ns = append(dbcData.Ns, strings.TrimSpace(line))
	}
}

func (p *Parser) parseBS(curIdx *int, buf []string) {
	subs := strings.Split(buf[*curIdx], ":")
	if len(subs) > 1 {
		dbcData.Bs = subs[1]
	}
}

func (p *Parser) parseBU(curIdx *int, buf []string) {
	subs := strings.Split(buf[*curIdx], ":")
	if len(subs) < 2 {
		return
	}

	bus := strings.Split(strings.TrimSpace(subs[1]), " ")
	for _, bu := range bus {
		dbcData.Bu = append(dbcData.Bu, bu)
	}
}

func (p *Parser) parseValTable(curIdx *int, buf []string) {
	subs := strings.Split(strings.TrimSpace(buf[*curIdx]), "\"")
	if len(subs) < 2 {
		return
	}

	sliceVal0 := regexp.MustCompile(" |:").Split(strings.TrimSpace(subs[0]), -1)
	valList := sliceVal0
	valList = append(valList, subs[1:]...)

	valTable := ValTableVO{"", make(map[string]string)}
	valTable.SignalName = valList[0]
	for i := 2; i+1 < len(valList); i = i + 2 {
		key := strings.TrimSpace(valList[i])
		if _, err := strconv.ParseInt(key, 10, 64); err != nil {
			return
		}

		valTable.DefinedMap[key] = valList[i+1]
	}

	dbcData.ValTableVOList = append(dbcData.ValTableVOList, valTable)
}

func (p *Parser) parseBO(curIdx *int, buf []string) {
	subs := regexp.MustCompile(" |:").Split(buf[*curIdx], -1)
	if len(subs) < 5 {
		return
	}

	for i := 0; i < len(subs); i++ {
		if "" == subs[i] {
			tmp := subs[0:i]
			tmp = append(tmp, subs[i+1:]...)
			subs = tmp
		}
	}

	var boVO BoVO
	boVO.CanId, _ = strconv.ParseUint(subs[1], 10, 64)
	boVO.CanName = subs[2]
	boVO.DataLenth, _ = strconv.ParseUint(subs[3], 10, 64)
	boVO.Transmitter = subs[4]
	boVO.DbcContent = buf[*curIdx]

	if dbcData.BoVoMap == nil {
		dbcData.BoVoMap = make(map[uint64]*BoVO)
	}
	dbcData.BoVoMap[boVO.CanId] = &boVO
	// 下一行
	(*curIdx)++
	p.parseSG(curIdx, buf, &boVO)
}

func (p *Parser) parseSG(curIdx *int, buf []string, boVO *BoVO) {
	const expr = "^SG_(.*):(.*)\\|(.*)@(\\d)([+\\-]).*\\((.*),(.*)\\).*\\[(.*)\\|(.*)].*\"(.*)\"(.*)$"
	r := regexp.MustCompile(expr)
	for ; *curIdx < len(buf); (*curIdx)++ {
		line := strings.TrimSpace(buf[*curIdx])
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(line)), kSG) {
			(*curIdx)--
			return
		}

		if r.MatchString(line) {
			var matcheStr []string
			if matcheStr = r.FindStringSubmatch(line); matcheStr == nil {
				log.Warnln("Nothing is reg matched")
				continue
			}

			if len(matcheStr) < 12 {
				log.Errorf("Reg match count invalid, match count(%d)\n", len(matcheStr))
				return
			}
			var sgVO SgVO
			arr := strings.Split(matcheStr[1], "\\(")
			sgVO.SignalName = strings.TrimSpace(arr[0])
			if len(arr) > 1 {
				sgVO.SigTypeDefinition = arr[1][:len(arr[1])-1]
			}

			sgVO.StartBit, _ = strconv.Atoi(strings.TrimSpace(matcheStr[2]))
			sgVO.BitWidth, _ = strconv.Atoi(strings.TrimSpace(matcheStr[3]))
			sgVO.ByteOrder, _ = strconv.Atoi(strings.TrimSpace(matcheStr[4]))
			sgVO.ValueType = strings.TrimSpace(matcheStr[5])
			sgVO.Factor, _ = strconv.ParseFloat(strings.TrimSpace(matcheStr[6]), 64)
			sgVO.Offsets, _ = strconv.ParseInt(strings.TrimSpace(matcheStr[7]), 10, 64)
			sgVO.Min, _ = strconv.ParseFloat(strings.TrimSpace(matcheStr[8]), 64)
			sgVO.Max, _ = strconv.ParseFloat(strings.TrimSpace(matcheStr[9]), 64)
			sgVO.Unit = strings.TrimSpace(matcheStr[10])
			sgVO.Receiver = strings.TrimSpace(matcheStr[11])
			sgVO.DbcContent = line
			if sgVO.DefineMap == nil {
				sgVO.DefineMap = make(map[string]string)
			}

			if boVO.SgVoMap == nil {
				boVO.SgVoMap = make(map[string]*SgVO)
			}
			boVO.SgVoMap[sgVO.SignalName] = &sgVO
			boVO.OrderedSignals = append(boVO.OrderedSignals, sgVO.SignalName)
		} else {
			log.Errorf(line, " is not reg match")
		}
	}
}

func (p *Parser) parseCM(curIdx *int, buf []string) {
	curLine := buf[*curIdx]
	valarr2 := strings.SplitN(curLine, "\"", 2)
	valarr1 := strings.Split(strings.TrimSpace(valarr2[0]), " ")

	valArr := valarr1
	valArr = append(valArr, valarr2[1])

	var cmVO CmVO
	cmVO.DbcContent = curLine
	if len(valArr) < 2 {
		return
	}

	if kSG == valArr[1] {
		cmVO.CanId, _ = strconv.ParseUint(valArr[2], 10, 64)
		cmVO.ObjectType = valArr[1]
		cmVO.Name = valArr[3]
		cmVO.Comment = valArr[4]
	} else if kBO == valArr[1] {
		cmVO.CanId, _ = strconv.ParseUint(valArr[2], 10, 64)
		cmVO.ObjectType = valArr[1]
		cmVO.Name = valArr[3]
		cmVO.Comment = valArr[4]
	} else if kBU == valArr[1] {
		cmVO.ObjectType = valArr[1]
		cmVO.Name = valArr[2]
		cmVO.Comment = valArr[3]
	}

	// 处理后缀
	if strings.HasSuffix(cmVO.Comment, "\";") {
		cmVO.Comment = cmVO.Comment[:len(cmVO.Comment)-len("\";")]
	}

	// 描述为第一段
	if cmVO.Comment != "" {
		cmArr := strings.Split(cmVO.Comment, "[ \n]")
		if !isNumber(cmArr[0]) {
			cmVO.Desc = cmArr[0]
		}
	}

	if !strings.HasSuffix(curLine, "\";") {
		(*curIdx)++
		p.parseCmEnd(curIdx, buf, &cmVO)
	}

	dbcData.CmVOList = append(dbcData.CmVOList, cmVO)
}

func (p *Parser) parseCmEnd(curIdx *int, buf []string, cmVO *CmVO) {
	for ; *curIdx < len(buf); (*curIdx)++ {
		curLine := buf[*curIdx]
		cmVO.Comment = cmVO.Comment + " " + curLine
		cmVO.DbcContent = cmVO.DbcContent + " " + curLine
		if strings.HasSuffix(curLine, "\";") {
			return
		}
	}
}

func (p *Parser) parseVal(curIdx *int, buf []string) {
	curLine := buf[*curIdx]
	valArr1 := strings.Split(curLine, "\"")
	valArr2 := strings.Split(strings.TrimSpace(valArr1[0]), " ")

	valArr := valArr2
	valArr = append(valArr, valArr1[1:]...)

	var valVO ValVO
	valVO.DefineMap = make(map[string]string)
	valVO.CanId, _ = strconv.ParseUint(strings.TrimSpace(valArr[1]), 10, 64)
	valVO.SignalName = strings.TrimSpace(valArr[2])
	valVO.DbcContent = curLine
	for i := 3; i+1 < len(valArr); i = i + 2 {
		key := strings.TrimSpace(valArr[i])
		if isNumber(key) {
			valVO.DefineMap[key] = strings.TrimSpace(valArr[i+1])
		}
	}

	dbcData.ValVOList = append(dbcData.ValVOList, valVO)
}

func isNumber(str string) bool {
	if str == "" {
		return false
	}

	for _, x := range []rune(str) {
		if !unicode.IsNumber(x) {
			return false
		}
	}
	return true
}

// Err returns the first non-EOF error that was encountered by the Scanner.
func (p *Parser) Err() error {
	if p.err == io.EOF {
		return nil
	}
	return p.err
}

// setErr records the first error encountered.
func (p *Parser) setErr(err error) {
	if p.err == nil || p.err == io.EOF {
		p.err = err
	}
}

func BoVoMap() map[uint64]*BoVO {
	return dbcData.BoVoMap
}
