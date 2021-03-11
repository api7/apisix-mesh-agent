package sidecar

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/api7/apisix-mesh-agent/pkg/log"

	"github.com/stretchr/testify/assert"
)

func TestConfigRender(t *testing.T) {
	assert.Nil(t, os.Mkdir("./testdata/conf", 0755))
	defer func() {
		assert.Nil(t, os.RemoveAll("./testdata/conf"))
	}()
	ar := &apisixRunner{
		config: &apisixConfig{
			SSLPort:       9443,
			NodeListen:    9080,
			GRPCListen:    "127.0.0.1:2379",
			EtcdKeyPrefix: "/apisix",
		},
		runArgs: []string{"start"},
		home:    "./testdata",
	}
	err := ar.renderConfig()
	assert.Nil(t, err)

	data, err := ioutil.ReadFile("./testdata/conf/config-default.yaml")
	assert.Nil(t, err)
	assert.Contains(t, string(data), "node_listen: 9080")
	assert.Contains(t, string(data), "prefix: \"/apisix\"")
	assert.Contains(t, string(data), "- \"http://127.0.0.1:2379\"")
}

func TestApisixRunner(t *testing.T) {
	assert.Nil(t, os.Mkdir("./testdata/conf", 0755))
	defer func() {
		assert.Nil(t, os.RemoveAll("./testdata/conf"))
	}()
	ar := &apisixRunner{
		logger: log.DefaultLogger,
		config: &apisixConfig{
			SSLPort:       9443,
			NodeListen:    9080,
			GRPCListen:    "127.0.0.1:2379",
			EtcdKeyPrefix: "/apisix",
		},
		runArgs: []string{"3600"},
		home:    "./testdata",
		bin:     "sleep",
	}
	assert.Nil(t, ar.run())
	pid := ar.process.Pid
	assert.NotEqual(t, pid, 0)
	ar.shutdown()
}
