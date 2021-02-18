// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
