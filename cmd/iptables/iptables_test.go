package iptables

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaptureAllInboundTraffic(t *testing.T) {
	f, err := ioutil.TempFile("./", "iptables.*")
	assert.Nil(t, err)
	defer func() {
		assert.Nil(t, f.Close())
		assert.Nil(t, os.Remove(f.Name()))
	}()
	rawStdout := os.Stdout
	os.Stdout = f
	cmd := NewSetupCommand()
	cmd.SetArgs([]string{
		"--apisix-port",
		"9080",
		"--dry-run",
	})
	err = cmd.Execute()
	os.Stdout = rawStdout
	assert.Nil(t, err)
	expect := []string{
		"iptables -t nat -N APISIX_REDIRECT",
		"iptables -t nat -N APISIX_INBOUND_REDIRECT",
		"iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080",
		"iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081",
	}
	data, err := ioutil.ReadFile(f.Name())
	assert.Nil(t, err)
	actual := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Equal(t, expect, actual)
}

func TestCaptureSelectedInboundTraffic(t *testing.T) {
	f, err := ioutil.TempFile("./", "iptables.*")
	assert.Nil(t, err)
	defer func() {
		assert.Nil(t, f.Close())
		assert.Nil(t, os.Remove(f.Name()))
	}()
	rawStdout := os.Stdout
	os.Stdout = f
	cmd := NewSetupCommand()
	cmd.SetArgs([]string{
		"--apisix-port",
		"9080",
		"--inbound-ports",
		"80,443,53",
		"--dry-run",
	})
	err = cmd.Execute()
	os.Stdout = rawStdout
	assert.Nil(t, err)
	expect := []string{
		"iptables -t nat -N APISIX_REDIRECT",
		"iptables -t nat -N APISIX_INBOUND_REDIRECT",
		"iptables -t nat -N APISIX_INBOUND",
		"iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080",
		"iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081",
		"iptables -t nat -A PREROUTING -p tcp -j APISIX_INBOUND",
		"iptables -t nat -A APISIX_INBOUND -p tcp --dport 80 -j APISIX_INBOUND_REDIRECT",
		"iptables -t nat -A APISIX_INBOUND -p tcp --dport 443 -j APISIX_INBOUND_REDIRECT",
		"iptables -t nat -A APISIX_INBOUND -p tcp --dport 53 -j APISIX_INBOUND_REDIRECT",
	}
	data, err := ioutil.ReadFile(f.Name())
	assert.Nil(t, err)
	actual := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Equal(t, expect, actual)

}

func TestCaptureOutboundTraffic(t *testing.T) {
	f, err := ioutil.TempFile("./", "iptables.*")
	assert.Nil(t, err)
	defer func() {
		assert.Nil(t, f.Close())
		assert.Nil(t, os.Remove(f.Name()))
	}()
	rawStdout := os.Stdout
	os.Stdout = f
	cmd := NewSetupCommand()
	cmd.SetArgs([]string{
		"--apisix-port",
		"9080",
		"--outbound-ports",
		"80,443",
		"--dry-run",
	})
	err = cmd.Execute()
	os.Stdout = rawStdout
	assert.Nil(t, err)
	expect := []string{
		"iptables -t nat -N APISIX_REDIRECT",
		"iptables -t nat -N APISIX_INBOUND_REDIRECT",
		"iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080",
		"iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081",
		"iptables -t nat -A OUTPUT -p tcp --dport 80 -j APISIX_REDIRECT",
		"iptables -t nat -A OUTPUT -p tcp --dport 443 -j APISIX_REDIRECT",
	}
	data, err := ioutil.ReadFile(f.Name())
	assert.Nil(t, err)
	actual := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Equal(t, expect, actual)
}

func TestCaptureBothInboundAndOutboundTraffic(t *testing.T) {
	f, err := ioutil.TempFile("./", "iptables.*")
	assert.Nil(t, err)
	defer func() {
		assert.Nil(t, f.Close())
		assert.Nil(t, os.Remove(f.Name()))
	}()
	rawStdout := os.Stdout
	os.Stdout = f
	cmd := NewSetupCommand()
	cmd.SetArgs([]string{
		"--apisix-port",
		"9080",
		"--outbound-ports",
		"80,443",
		"--inbound-ports",
		"*",
		"--dry-run",
	})
	err = cmd.Execute()
	os.Stdout = rawStdout
	assert.Nil(t, err)
	expect := []string{
		"iptables -t nat -N APISIX_REDIRECT",
		"iptables -t nat -N APISIX_INBOUND_REDIRECT",
		"iptables -t nat -N APISIX_INBOUND",
		"iptables -t nat -A APISIX_REDIRECT -p tcp -j REDIRECT --to-ports 9080",
		"iptables -t nat -A APISIX_INBOUND_REDIRECT -p tcp -j REDIRECT --to-ports 9081",
		"iptables -t nat -A PREROUTING -p tcp -j APISIX_INBOUND",
		"iptables -t nat -A APISIX_INBOUND -p tcp --dport 22 -j RETURN",
		"iptables -t nat -A APISIX_INBOUND -p tcp -j APISIX_INBOUND_REDIRECT",
		"iptables -t nat -A OUTPUT -p tcp --dport 80 -j APISIX_REDIRECT",
		"iptables -t nat -A OUTPUT -p tcp --dport 443 -j APISIX_REDIRECT",
	}
	data, err := ioutil.ReadFile(f.Name())
	assert.Nil(t, err)
	actual := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Equal(t, expect, actual)
}
