package sidecar

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/log"
)

type apisixRunner struct {
	home    string
	bin     string
	logger  *log.Logger
	done    chan struct{}
	process *os.Process
}

func (ar *apisixRunner) run(stop chan struct{}) error {
	errCh := make(chan error)
	cmd := exec.Command(ar.bin, "start")
	go func() {
		if err := cmd.Run(); err != nil {
			ar.logger.Fatalw("apisix running failure",
				zap.Error(err),
				zap.String("bin", ar.bin),
			)
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-time.After(2 * time.Second):
		ar.process = cmd.Process
		break
	}
	ar.logger.Infow("launch apisix",
		zap.Int("master_pid", cmd.Process.Pid),
	)
	return nil
}

func (ar *apisixRunner) shutdown() {
	if ar.process == nil {
		return
	}
	ar.logger.Info("closing apisix")
	if err := ar.process.Signal(syscall.SIGINT); err != nil {
		ar.logger.Fatalw("failed to send SIGINT signal to apisix master process",
			zap.Int("master_pid", ar.process.Pid),
		)
	}
}
