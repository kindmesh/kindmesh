package netdevice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDevice(t *testing.T) {
	err := EnsureDevice("bridge0")
	assert.Nil(t, err)

	ip1 := "169.254.200.10"
	err = AddAddr(ip1)
	assert.Nil(t, err)

	ret, err := ListAddr()
	assert.Nil(t, err)
	assert.Equal(t, ip1, ret[0])

	err = DelAddr(ip1)
	assert.Nil(t, err)

	ret, _ = ListAddr()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(ret))
}
