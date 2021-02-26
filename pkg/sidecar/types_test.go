package sidecar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/id"
)

func TestSidecarRun(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.XDSWatchFiles = append(cfg.XDSWatchFiles, "testdata/cluster.json")
	cfg.GRPCListen = "127.0.0.1:10002"
	s, err := NewSidecar(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, s)

	stop := make(chan struct{})
	finishCh := make(chan struct{})
	go func() {
		err := s.Run(stop)
		assert.Nil(t, err)
		close(finishCh)
	}()

	time.Sleep(time.Second)
	close(stop)
	<-finishCh

	ups, err := s.cache.Upstream().Get(id.GenID("httpbin.default.svc.cluster.local"))
	assert.Nil(t, err)
	assert.Equal(t, ups.Name, "httpbin.default.svc.cluster.local")
	assert.Len(t, ups.Nodes, 0)

}
