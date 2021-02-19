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

package xds

import (
	"github.com/fsnotify/fsnotify"
	"honnef.co/go/tools/config"

	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/provisioner"
)

type xdsFileWatcherProvisioner struct {
	logger  *log.Logger
	watcher *fsnotify.Watcher
}

// NewXDSProvisionerFromFiles creates a files backed Provisioner, it watches
// on the given files/directories, files will be parsed into xDS objects,
// invalid items will be ignored but leave with a log.
func NewXDSProvisionerFromFiles(cfg *config.Config) (provisioner.Provisioner, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	p := &xdsFileWatcherProvisioner{
		watcher: watcher,
	}
	return p, nil
}
