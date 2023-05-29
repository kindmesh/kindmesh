package processor

import (
	"testing"

	"github.com/kindmesh/kindmesh/internal/watch/netdevice"
	"github.com/stretchr/testify/assert"
)

func TestMalloctor(t *testing.T) {
	netdevice.EnsureDevice("bridge0")
	m := newDefaultMalloctor()
	ret, err := m.AllocateForNames(map[string]bool{
		"a": true,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ret))

	ret, err = m.AllocateForNames(map[string]bool{
		"a": true,
		"b": true,
	})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(ret))
	ret, err = m.AllocateForNames(map[string]bool{
		"b": true,
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ret))
	ret, err = m.AllocateForNames(map[string]bool{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(ret))
}
