package mapmap

import "sync"

type MapMap struct {
	mtx sync.RWMutex
	kkv map[any]map[any]any
}

func NewMapMap() *MapMap {
	return &MapMap{
		kkv: map[any]map[any]any{},
	}
}

func (mm *MapMap) MSet(key, subkey, value any) {
	mm.mtx.Lock()
	defer mm.mtx.Unlock()

	kv, ok := mm.kkv[key]
	if !ok {
		kv = map[any]any{subkey: value}
		mm.kkv[key] = kv
		return
	}

	kv[subkey] = value
	mm.kkv[key] = kv
}

func (mm *MapMap) MGet(key, subkey any) any {
	mm.mtx.RLock()
	defer mm.mtx.RUnlock()

	kv, ok := mm.kkv[key]
	if !ok {
		return nil
	}

	return kv[subkey]
}

func (mm *MapMap) MGetAll(key any) []any {
	mm.mtx.RLock()
	defer mm.mtx.RUnlock()

	kv, ok := mm.kkv[key]
	if !ok {
		return nil
	}

	vs := []any{}
	for _, v := range kv {
		vs = append(vs, v)
	}
	return vs
}

func (mm *MapMap) MDel(key, subkey any) {
	mm.mtx.Lock()
	defer mm.mtx.Unlock()

	kv, ok := mm.kkv[key]
	if !ok {
		return
	}
	delete(kv, subkey)
}

func (mm *MapMap) MDelAll(key any) {
	mm.mtx.Lock()
	defer mm.mtx.Unlock()

	delete(mm.kkv, key)
}
