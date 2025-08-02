package whitelist

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	"CANAdapter/base"
	"CANAdapter/dbc"
)

var log = base.Logger

const (
	OK uint = iota
	ReadBodyError
	ParseJsonError
	InvalidAction
	WrongHttpMethod
)

// Action
const (
	Do_ResetWith int = iota + 1
	Do_Add
	Do_Delete
)

var (
	AsyncSaveChan = make(chan bool)
	WhiteListCode = make(map[uint]string)
)

func init() {
	WhiteListCode[OK] = "OK"
	WhiteListCode[ReadBodyError] = "Read body error"
	WhiteListCode[ParseJsonError] = "Parse json error"
	WhiteListCode[InvalidAction] = "Invalid action"
	WhiteListCode[WrongHttpMethod] = "Wrong http method, should use POST"
}

type WhiteListRsp struct {
	StatusCode uint   `json:"statusCode"`
	Reason     string `json:"reason"`
}

type WhiteListReq struct {
	TaskId    int                 `json:"taskId"`
	Action    int                 `json:"action"`
	CanList   map[string][]string `json:"canList"`
	TimeStamp string              `json:"timeStamp"`
}

type WhiteListMap map[uint64]map[string]bool

type WhiteList struct {
	mu           sync.Mutex
	whiteListMap WhiteListMap
	enable       bool
}

var g_WhiteList = &WhiteList{
	whiteListMap: make(WhiteListMap),
}

func SetEnableFlag(enable bool) {
	g_WhiteList.setEnableFlag(enable)
}

func IsEnable() bool {
	return g_WhiteList.isEnable()
}

func QueryByCanId(canId uint64) bool {
	return g_WhiteList.queryByCanId(canId)
}

func QueryByCanIdAndSignal(canId uint64, signal string) bool {
	return g_WhiteList.queryByCanIdAndSignal(canId, signal)
}

// func ResetWith(req *WhiteListReq) {
// 	g_WhiteList.resetWith(req)
// }

// func Add(req *WhiteListReq) {
// 	g_WhiteList.add(req)
// }

// func Delete(req *WhiteListReq) {
// 	g_WhiteList.delete(req)
// }

func (w *WhiteList) setEnableFlag(enable bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.enable = enable
}

func (w *WhiteList) isEnable() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.enable
}

func (w *WhiteList) queryByCanId(canId uint64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.whiteListMap[canId]; !ok {
		return false
	}
	return true
}

func (w *WhiteList) queryByCanIdAndSignal(canId uint64, signal string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if signals, ok := w.whiteListMap[canId]; ok {
		if _, ok := signals[signal]; ok {
			return true
		}
	}

	return false
}

func (w *WhiteList) resetWith(req *WhiteListReq) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// 重置清空map
	w.whiteListMap = WhiteListMap{}
	w.innerAdd(req)
}

func (w *WhiteList) add(req *WhiteListReq) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.innerAdd(req)
}

func (w *WhiteList) innerAdd(req *WhiteListReq) {
	for strCanId, vSignals := range req.CanList {
		canId, err := strconv.ParseUint(strCanId, 10, 64)
		if err != nil {
			log.Errorln(err)
			continue
		}

		signals := w.whiteListMap[canId]
		if signals == nil {
			signals = make(map[string]bool)
			w.whiteListMap[canId] = signals
		}

		if len(vSignals) == 1 && "*" == vSignals[0] {
			var ok bool
			var boVO *dbc.BoVO
			if boVO, ok = dbc.BoVoMap()[canId]; !ok {
				log.Errorf("No dbc data !!! canId(%d)", canId)
				continue
			}

			for _, signal := range boVO.OrderedSignals {
				signals[signal] = true
			}
		} else {
			for _, signal := range vSignals {
				signals[signal] = true
			}
		}
	}
}

func (w *WhiteList) delete(req *WhiteListReq) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for strCanId, vSignals := range req.CanList {
		canId, err := strconv.ParseUint(strCanId, 10, 64)
		if err != nil {
			log.Errorln(err)
			continue
		}

		if signals, ok := w.whiteListMap[canId]; ok {
			// canId存在
			if len(vSignals) == 1 && "*" == vSignals[0] {
				var ok bool
				var boVO *dbc.BoVO
				if boVO, ok = dbc.BoVoMap()[canId]; !ok {
					log.Errorf("No dbc data !!! canId(%d)", canId)
					continue
				}

				for _, signal := range boVO.OrderedSignals {
					delete(signals, signal)
				}
			} else {
				for _, signal := range vSignals {
					delete(signals, signal)
				}
			}

			if len(signals) <= 0 {
				delete(w.whiteListMap, canId)
			}
		}
	}
}

func SetWhiteList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		rspByCode(w, WrongHttpMethod, http.StatusMethodNotAllowed)
		return
	}

	all, err := io.ReadAll(r.Body)
	if err != nil {
		rspByCode(w, ReadBodyError, http.StatusInternalServerError)
		return
	}

	req := WhiteListReq{}
	err = json.Unmarshal(all, &req)
	if err != nil {
		rspByCode(w, ParseJsonError, http.StatusUnprocessableEntity)
		return
	}

	switch req.Action {
	case Do_ResetWith:
		g_WhiteList.resetWith(&req)
	case Do_Add:
		g_WhiteList.add(&req)
	case Do_Delete:
		g_WhiteList.delete(&req)
	default:
		{
			rspByCode(w, InvalidAction, http.StatusUnprocessableEntity)
			return
		}
	}

	AsyncSaveChan <- true
	rspByCode(w, OK, http.StatusOK)
}

func rspByCode(w http.ResponseWriter, errCode uint, statusCode int) {
	rsp, _ := toJsonRsp(errCode)
	w.WriteHeader(statusCode)
	w.Write(rsp)
}

func toJsonRsp(errCode uint) ([]byte, error) {
	rsp := &WhiteListRsp{errCode, WhiteListCode[errCode]}
	jData, err := json.Marshal(rsp)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}

	jData = append(jData, '\n')
	return jData, nil
}

func Init(whiteListFile string, wg *sync.WaitGroup, enable bool) (bool, error) {
	g_WhiteList.setEnableFlag(enable)

	ok, err := g_WhiteList.loadFromFile(whiteListFile)
	if err != nil {
		return ok, err
	}

	wg.Add(1)
	go g_WhiteList.asyncSave2WhiteList(whiteListFile, wg)

	return true, nil
}

func (w *WhiteList) loadFromFile(whiteListFile string) (bool, error) {
	if len(whiteListFile) <= 0 {
		return false, errors.New("WhiteList filename is empty")
	}

	file, err := os.OpenFile(whiteListFile, os.O_RDONLY|os.O_CREATE, 0o666)
	if err != nil {
		return false, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&w.whiteListMap)
	if err != nil {
		if err != io.EOF {
			return false, err
		}
	}

	return true, nil
}

func (w *WhiteList) asyncSave2WhiteList(whiteListFile string, wg *sync.WaitGroup) {
	defer wg.Done()

	file, err := os.OpenFile(whiteListFile, os.O_WRONLY|os.O_CREATE, 0o666)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	for range AsyncSaveChan {
		buf, err := w.Marshal(w.whiteListMap)
		if err != nil {
			log.Errorln(err)
			continue
		}

		err = file.Truncate(0)
		if err != nil {
			log.Fatalln(err)
			continue
		}

		n, err := file.WriteAt(buf, 0)
		if err != nil {
			log.Errorf("Write (%s) failed! reason:(%s), has written (%d) bytes \n", whiteListFile, err.Error(), n)
		}

		log.Debugf("%s\nWrite (%s) ok! has written (%d) bytes\n", string(buf), whiteListFile, n)
	}
}

func (w *WhiteList) Marshal(whiteListMap WhiteListMap) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return json.Marshal(whiteListMap)
}
