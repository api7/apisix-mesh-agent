package iptables

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanupIptables(t *testing.T) {
	f, err := ioutil.TempFile("./", "iptables-cleanup.*")
	assert.Nil(t, err)
	defer func() {
		assert.Nil(t, f.Close())
		assert.Nil(t, os.Remove(f.Name()))
	}()
	rawStdout := os.Stdout
	os.Stdout = f
	cleanup(true)
	os.Stdout = rawStdout

	data, err := ioutil.ReadFile(f.Name())
	assert.Nil(t, err)

	expect := `iptables -t nat -D PREROUTING -p tcp -j APISIX_INBOUND
iptables -t nat -D OUTPUT -p tcp -j OUTPUT
iptables -t nat -F APISIX_INBOUND
iptables -t nat -X APISIX_INBOUND
iptables -t nat -F OUTPUT
iptables -t nat -X OUTPUT
iptables -t nat -F APISIX_REDIRECT
iptables -t nat -X APISIX_REDIRECT
iptables -t nat -F APISIX_INBOUND_REDIRECT
iptables -t nat -X APISIX_INBOUND_REDIRECT
`
	assert.Equal(t, expect, string(data))
}
