package rwslice

import (
	"sync"

	"CANAdapter/can"
)

type RWSlice struct {
	sync.Mutex
	timeQueue []can.PDU
}

// 新建一个RWSlice
func NewRWSlice(n int) *RWSlice {
	return &RWSlice{
		timeQueue: make([]can.PDU, n),
	}
}

func NewEmptyRWSlice() *RWSlice {
	return NewRWSlice(0)
}

func (m *RWSlice) Index(k uint32) *can.PDU { // 从slice中读取一个值
	m.Lock()
	defer m.Unlock()
	v := m.timeQueue[k] // 在锁的保护下从slice中读取
	return &v
}

func (m *RWSlice) Append(v *can.PDU) { // 设置一个键值对
	m.Lock() // 锁保护
	defer m.Unlock()
	m.timeQueue = append(m.timeQueue, *v)
}

func (m *RWSlice) Shrink(begin, end uint32) []can.PDU { // 删除一个键
	m.Lock() // 锁保护
	defer m.Unlock()
	return m.timeQueue[begin:end]
}

func (m *RWSlice) Clear() { // 删除一个键
	m.Lock() // 锁保护
	defer m.Unlock()
	m.timeQueue = []can.PDU{}
}

func (m *RWSlice) Len() int { // slice的长度
	m.Lock() // 锁保护
	defer m.Unlock()
	return len(m.timeQueue)
}

func (m *RWSlice) Each(f func(k uint64, v *can.PDU) bool) { // 遍历slice
	m.Lock() // 遍历期间一直持有读锁
	defer m.Unlock()

	for k, v := range m.timeQueue {
		if !f(uint64(k), &v) {
			return
		}
	}
}
