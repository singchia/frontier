package mapmap

import "sync"

// 1 value: n keys
// 1 key: 1 value
type BiMap struct {
	mtx sync.RWMutex
	kv  map[any]any
	vk  map[any]map[any]struct{}
}

func NewBiMap() *BiMap {
	return &BiMap{
		kv: map[any]any{},
		vk: map[any]map[any]struct{}{},
	}
}

func (bm *BiMap) Set(key, value any) {
	bm.mtx.Lock()
	defer bm.mtx.Unlock()

	// Remove the previous reverse entry when a key changes ownership.
	if old, ok := bm.kv[key]; ok && old != value {
		delete(bm.vk[old], key)
		if len(bm.vk[old]) == 0 {
			delete(bm.vk, old)
		}
	}
	bm.kv[key] = value
	ks, ok := bm.vk[value]
	if !ok {
		ks = map[any]struct{}{}
	}
	ks[key] = struct{}{}
	bm.vk[value] = ks
}

func (bm *BiMap) GetValue(key any) (any, bool) {
	bm.mtx.RLock()
	defer bm.mtx.RUnlock()

	value, ok := bm.kv[key]
	return value, ok
}

func (bm *BiMap) Del(key any) bool {
	bm.mtx.Lock()
	defer bm.mtx.Unlock()

	value, ok := bm.kv[key]
	if !ok {
		return false
	}
	delete(bm.kv, key)
	ks, ok := bm.vk[value]
	if ok {
		delete(ks, key)
		if len(ks) == 0 {
			delete(bm.vk, value)
		} else {
			bm.vk[value] = ks
		}
	}
	return true
}

func (bm *BiMap) GetKeys(value any) ([]any, bool) {
	bm.mtx.RLock()
	defer bm.mtx.RUnlock()

	ks, ok := bm.vk[value]
	if !ok {
		return nil, false
	}
	slice := []any{}
	for k := range ks {
		slice = append(slice, k)
	}
	return slice, true
}

func (bm *BiMap) DelValue(value any) bool {
	bm.mtx.Lock()
	defer bm.mtx.Unlock()

	ks, ok := bm.vk[value]
	if !ok {
		return false
	}
	delete(bm.vk, value)
	for k := range ks {
		delete(bm.kv, k)
	}
	return true
}

// CompareAndDelete removes a mapping only while it still points at value.
func (bm *BiMap) CompareAndDelete(key, value any) bool {
	bm.mtx.Lock()
	defer bm.mtx.Unlock()
	current, ok := bm.kv[key]
	if !ok || current != value {
		return false
	}
	delete(bm.kv, key)
	delete(bm.vk[value], key)
	if len(bm.vk[value]) == 0 {
		delete(bm.vk, value)
	}
	return true
}
