package mapmap

import "sync"

type BiMap struct {
	mtx sync.RWMutex
	kv  map[any]any
	vk  map[any]any
}

func NewBiMap() *BiMap {
	return &BiMap{
		kv: map[any]any{},
		vk: map[any]any{},
	}
}

func (bm *BiMap) Set(key, value any) {
	bm.mtx.Lock()
	defer bm.mtx.Unlock()

	bm.kv[key] = value
	bm.vk[value] = key
}
