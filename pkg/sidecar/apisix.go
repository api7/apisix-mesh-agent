package sidecar

import (
	"bytes"
	_ "embed"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"text/template"
	"time"

	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/log"
)

var (
	//go:embed apisix/config.yaml
	_configYaml string
)

type apisixRunner struct {
	config  *apisixConfig
	home    string
	bin     string
	runArgs []string
	logger  *log.Logger
	done    chan struct{}
	process *os.Process
}

type apisixConfig struct {
	SSLPort       int
	NodeListen    int
	GRPCListen    string
	EtcdKeyPrefix string
}

func (ar *apisixRunner) run() error {
	if err := ar.renderConfig(); err != nil {
		return err
	}

	errCh := make(chan error)
	cmd := exec.Command(ar.bin, ar.runArgs...)
	go func() {
		stderr := bytes.NewBuffer(nil)
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			ar.logger.Fatalw("apisix running failure",
				zap.Error(err),
				zap.String("bin", ar.bin),
				zap.String("stderr", stderr.String()),
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

func (ar *apisixRunner) renderConfig() error {
	temp, err := template.New("apisix-config").Parse(_configYaml)
	if err != nil {
		return err
	}
	var output bytes.Buffer
	if err := temp.Execute(&output, ar.config); err != nil {
		return err
	}
	filename := filepath.Join(ar.home, "conf", "config-default.yaml")
	if err := ioutil.WriteFile(filename, output.Bytes(), 0644); err != nil {
		return err
	}
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
