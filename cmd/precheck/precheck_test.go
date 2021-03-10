package precheck

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckBin(t *testing.T) {
	var buffer strings.Builder

	assert.Equal(t, checkBin(&buffer, "./not_a_file"), false)
	expect := "checking apisix binary path ./not_a_file ... stat ./not_a_file: no such file or directory\n"
	assert.Equal(t, expect, buffer.String())

	buffer.Reset()
	assert.Equal(t, checkBin(&buffer, "./precheck.go"), true)
	expect = "checking apisix binary path ./precheck.go ... found\n"
	assert.Equal(t, expect, buffer.String())
}

func TestCheckHome(t *testing.T) {
	var buffer strings.Builder

	assert.Equal(t, checkHome(&buffer, "./not_a_file"), false)
	expect := "checking apisix home path ./not_a_file ... stat ./not_a_file: no such file or directory\n"
	assert.Equal(t, expect, buffer.String())

	buffer.Reset()
	assert.Equal(t, checkHome(&buffer, "./precheck.go"), false)
	expect = "checking apisix home path ./precheck.go ... not a directory\n"
	assert.Equal(t, expect, buffer.String())

	buffer.Reset()
	assert.Equal(t, checkHome(&buffer, "../"), true)
	expect = "checking apisix home path ../ ... found\n"
	assert.Equal(t, expect, buffer.String())
}

func TestCheckIptables(t *testing.T) {
	var buffer strings.Builder

	assert.Equal(t, checkIptables(&buffer), false)
	expect := "checking iptables, table: nat, chain: APISIX_INBOUND ... exec: \"iptables\": executable file not found in $PATH\n"
	assert.Equal(t, expect, buffer.String())

	_iptablesCmd = "true"
	buffer.Reset()
	assert.Equal(t, checkIptables(&buffer), true)
	expect = `checking iptables, table: nat, chain: APISIX_INBOUND ... found
checking iptables, table: nat, chain: OUTPUT ... found
checking iptables, table: nat, chain: APISIX_REDIRECT ... found
checking iptables, table: nat, chain: PREROUTING ... found
`
	assert.Equal(t, expect, buffer.String())
}

func TestCheck(t *testing.T) {
	_iptablesCmd = "true"

	f, err := ioutil.TempFile("./", "stdout.*")
	assert.Nil(t, err)
	defer func() {
		assert.Nil(t, f.Close())
		assert.Nil(t, os.Remove(f.Name()))
	}()
	raw := os.Stderr
	os.Stderr = f

	ok := check("./precheck.go", "../")
	os.Stderr = raw
	assert.Equal(t, ok, true)

	data, err := ioutil.ReadFile(f.Name())
	assert.Nil(t, err)
	expect := `checking apisix binary path ./precheck.go ... found
checking apisix home path ../ ... found
checking iptables, table: nat, chain: APISIX_INBOUND ... found
checking iptables, table: nat, chain: OUTPUT ... found
checking iptables, table: nat, chain: APISIX_REDIRECT ... found
checking iptables, table: nat, chain: PREROUTING ... found
`
	assert.Equal(t, expect, string(data))
}
