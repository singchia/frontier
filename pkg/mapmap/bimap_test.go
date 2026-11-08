package mapmap

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMovedKeySurvivesOldValueRemoval(t *testing.T) {
	m := NewBiMap()
	m.Set(uint64(1), "a")
	m.Set(uint64(2), "a")
	m.Set(uint64(1), "b")
	keys, ok := m.GetKeys("a")
	require.True(t, ok)
	require.ElementsMatch(t, []any{uint64(2)}, keys)
	require.False(t, m.CompareAndDelete(uint64(1), "a"))
	m.DelValue("a")
	v, ok := m.GetValue(uint64(1))
	require.True(t, ok)
	require.Equal(t, "b", v)
	_, ok = m.GetValue(uint64(2))
	require.False(t, ok)
	require.True(t, m.CompareAndDelete(uint64(1), "b"))
	_, ok = m.GetKeys("b")
	require.False(t, ok)
}
func TestConcurrentStaleInvalidation(t *testing.T) {
	m := NewBiMap()
	m.Set(uint64(1), "a")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); m.CompareAndDelete(uint64(1), "a") }()
	}
	m.Set(uint64(1), "b")
	wg.Wait()
	v, ok := m.GetValue(uint64(1))
	require.True(t, ok)
	require.Equal(t, "b", v)
}
