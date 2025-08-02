package rwmap

import (
	"sync"

	"CANAdapter/can"
)

type RWMap struct {
	sync.RWMutex
	m map[int64]any
}

// 新建一个RWMap
func NewRWMap(n int) *RWMap {
	return &RWMap{
		m: make(map[int64]any, n),
	}
}

func (m *RWMap) Get(key int64) (any, bool) { // 从map中读取一个值
	m.RLock()
	defer m.RUnlock()
	v, existed := m.m[key] // 在锁的保护下从map中读取
	return v, existed
}

func (m *RWMap) Set(key int64, v any) { // 设置一个键值对
	m.Lock() // 锁保护
	defer m.Unlock()
	m.m[key] = v
}

func (m *RWMap) Delete(key int64) { // 删除一个键
	m.Lock() // 锁保护
	defer m.Unlock()
	delete(m.m, key)
}

func (m *RWMap) Clear() { // 删除一个键
	m.Lock() // 锁保护
	defer m.Unlock()
	m.m = map[int64]any{}
}

func (m *RWMap) Len() int { // map的长度
	m.RLock() // 锁保护
	defer m.RUnlock()
	return len(m.m)
}

func (m *RWMap) Each(f func(key int64, v any) bool) { // 遍历map
	m.RLock() // 遍历期间一直持有读锁
	defer m.RUnlock()

	for key, v := range m.m {
		if !f(key, v) {
			return
		}
	}
}

// **************************************************
type RWPduMap struct {
	sync.RWMutex
	m map[uint32]can.PDU
}

// 新建一个RWPduMap
func NewRWPduMap(n int) *RWPduMap {
	return &RWPduMap{
		m: make(map[uint32]can.PDU, n),
	}
}

func (m *RWPduMap) Get(k uint32) (*can.PDU, bool) { // 从map中读取一个值
	m.RLock()
	defer m.RUnlock()
	v, existed := m.m[k] // 在锁的保护下从map中读取
	return &v, existed
}

func (m *RWPduMap) Set(k uint32, v *can.PDU) { // 设置一个键值对
	m.Lock() // 锁保护
	defer m.Unlock()
	m.m[k] = *v
}

func (m *RWPduMap) Delete(k uint32) { // 删除一个键
	m.Lock() // 锁保护
	defer m.Unlock()
	delete(m.m, k)
}

func (m *RWPduMap) Clear() { // 删除一个键
	m.Lock() // 锁保护
	defer m.Unlock()
	m.m = map[uint32]can.PDU{}
}

func (m *RWPduMap) Len() int { // map的长度
	m.RLock() // 锁保护
	defer m.RUnlock()
	return len(m.m)
}

func (m *RWPduMap) Each(f func(k uint32, v *can.PDU) bool) { // 遍历map
	m.RLock() // 遍历期间一直持有读锁
	defer m.RUnlock()

	for k, v := range m.m {
		if !f(k, &v) {
			return
		}
	}
}
