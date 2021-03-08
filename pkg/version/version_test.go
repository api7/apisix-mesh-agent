package version

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestVersion(t *testing.T) {
	_version = "x.y.z"
	_gitRevision = "9a8bc1dd"
	_timestamp = "1613616943"

	ver := String()
	expectedVersion := `Version: x.y.z
Git SHA: 9a8bc1dd
Go Version: %s
OS/Arch: %s/%s
Build Date: %s
`
	date := time.Unix(1613616943, 0)
	expectedVersion = fmt.Sprintf(expectedVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH, date.String())
	assert.Equal(t, expectedVersion, ver, "bad version")
}

func TestShort(t *testing.T) {
	_version = "1.1.1"
	assert.Equal(t, Short(), _version)
}
