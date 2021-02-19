package version

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"time"
)

var (
	// The following fields are populated at build time using -ldflags -X.
	_version     = "unknown"
	_gitRevision = "unknown"
	_timestamp   = "0"
)

// String returns a readable version info.
func String() string {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "Version: %s\n", _version)
	fmt.Fprintf(buf, "Git SHA: %s\n", _gitRevision)
	fmt.Fprintf(buf, "Go Version: %s\n", runtime.Version())
	fmt.Fprintf(buf, "OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	ts, err := strconv.ParseInt(_timestamp, 10, 32)
	if err != nil {
		fmt.Fprintln(buf, "Build Date: unknown")
	} else {
		date := time.Unix(ts, 0)
		fmt.Fprintf(buf, "Build Date: %s\n", date.String())
	}

	return buf.String()
}
