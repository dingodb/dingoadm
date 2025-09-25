/*
 *  Copyright (c) 2022 NetEase Inc.
 * 	Copyright (c) 2024 dingodb.com Inc.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

/*
 * Project: CurveAdm
 * Created Date: 2022-07-14
 * Author: Jingli Chen (Wine93)
 */

// __SIGN_BY_WINE93__

package checker

import (
	"github.com/dingodb/dingoadm/internal/configure/topology"
)

type (
	Address struct {
		Role string
		IP   string
		Port int
	}

	Directory struct {
		Type string
		Path string
	}
)

const (
	LOG_DIR  = "log_dir"
	DATA_DIR = "data_dir"
	CORE_DIR = "core_dir"

	ROLE_ETCD             = topology.ROLE_ETCD
	ROLE_MDS              = topology.ROLE_MDS
	ROLE_CHUNKSERVER      = topology.ROLE_CHUNKSERVER
	ROLE_SNAPSHOTCLONE    = topology.ROLE_SNAPSHOTCLONE
	ROLE_METASERVER       = topology.ROLE_METASERVER
	ROLE_MDS_V2           = topology.ROLE_MDS_V2
	ROLE_COORDINATOR      = topology.ROLE_COORDINATOR
	ROLE_STORE            = topology.ROLE_STORE
	ROLE_DINGODB_DOCUMENT = topology.ROLE_DINGODB_DOCUMENT
	ROLE_DINGODB_DISKANN  = topology.ROLE_DINGODB_DISKANN
	ROLE_DINGODB_INDEX    = topology.ROLE_DINGODB_INDEX
	ROLE_DINGODB_EXECUTOR = topology.ROLE_DINGODB_EXECUTOR
	ROLE_DINGODB_WEB      = topology.ROLE_DINGODB_WEB
	ROLE_DINGODB_PROXY    = topology.ROLE_DINGODB_PROXY
)

var (
	CONNECT = map[string][]string{
		ROLE_ETCD:          {ROLE_ETCD},
		ROLE_MDS:           {ROLE_MDS, ROLE_ETCD},
		ROLE_CHUNKSERVER:   {ROLE_CHUNKSERVER, ROLE_MDS},
		ROLE_SNAPSHOTCLONE: {ROLE_SNAPSHOTCLONE},
		ROLE_METASERVER:    {ROLE_METASERVER, ROLE_MDS},
	}
)

/*
 * etcd -> { etcd }
 * mds -> { mds, etcd }
 * chunkserver -> { chunkserver, mds }
 * snapshotclone -> { snapshotclone }
 * metaserver -> { metaserver, mds }
 */
func getServiceConnectAddress(from *topology.DeployConfig, dcs []*topology.DeployConfig) []Address {
	m := map[string]bool{}
	for _, role := range CONNECT[from.GetRole()] {
		m[role] = true
	}

	address := []Address{}
	for _, to := range dcs {
		if from.GetId() == to.GetId() {
			continue
		} else if _, ok := m[to.GetRole()]; !ok {
			continue
		}

		address = append(address, getServiceListenAddresses(to)...)
	}
	return address
}

func getServiceListenAddresses(dc *topology.DeployConfig) []Address {
	address := []Address{}

	switch dc.GetRole() {
	case ROLE_ETCD:
		address = append(address, Address{
			Role: ROLE_ETCD,
			IP:   dc.GetListenIp(),
			Port: dc.GetListenPort(),
		})
		address = append(address, Address{
			Role: ROLE_ETCD,
			IP:   dc.GetListenIp(),
			Port: dc.GetListenClientPort(),
		})

	case ROLE_MDS:
		address = append(address, Address{
			Role: ROLE_MDS,
			IP:   dc.GetListenIp(),
			Port: dc.GetListenPort(),
		})
		address = append(address, Address{
			Role: ROLE_MDS,
			IP:   dc.GetListenIp(),
			Port: dc.GetListenDummyPort(),
		})

	case ROLE_CHUNKSERVER:
		address = append(address, Address{
			Role: ROLE_CHUNKSERVER,
			IP:   dc.GetListenIp(),
			Port: dc.GetListenPort(),
		})
		if dc.GetEnableExternalServer() {
			address = append(address, Address{
				Role: ROLE_CHUNKSERVER,
				IP:   dc.GetListenExternalIp(),
				Port: dc.GetListenExternalPort(),
			})
		}

	case ROLE_SNAPSHOTCLONE:
		address = append(address, Address{
			Role: ROLE_SNAPSHOTCLONE,
			IP:   dc.GetListenIp(),
			Port: dc.GetListenPort(),
		})
		address = append(address, Address{
			Role: ROLE_SNAPSHOTCLONE,
			IP:   dc.GetListenIp(),
			Port: dc.GetListenDummyPort(),
		})
		address = append(address, Address{
			Role: ROLE_SNAPSHOTCLONE,
			IP:   dc.GetListenIp(),
			Port: dc.GetListenProxyPort(),
		})

	case ROLE_METASERVER:
		address = append(address, Address{
			Role: ROLE_METASERVER,
			IP:   dc.GetListenIp(),
			Port: dc.GetListenPort(),
		})
		if dc.GetEnableExternalServer() {
			address = append(address, Address{
				Role: ROLE_METASERVER,
				IP:   dc.GetListenExternalIp(),
				Port: dc.GetListenExternalPort(),
			})
		}

	case ROLE_MDS_V2:
		address = append(address, Address{
			Role: ROLE_MDS_V2,
			IP:   dc.GetListenIp(),
			// Port: dc.GetListenPort(),
			Port: dc.GetDingoServerPort(),
		})

	case ROLE_COORDINATOR:
		address = append(address, Address{
			Role: ROLE_COORDINATOR,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoServerPort(),
		})
		address = append(address, Address{
			Role: ROLE_COORDINATOR,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoStoreRaftPort(),
		})

	case ROLE_STORE:
		address = append(address, Address{
			Role: ROLE_STORE,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoServerPort(),
		})
		address = append(address, Address{
			Role: ROLE_STORE,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoStoreRaftPort(),
		})
	case ROLE_DINGODB_DOCUMENT:
		address = append(address, Address{
			Role: ROLE_DINGODB_DOCUMENT,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoServerPort(),
		})
		address = append(address, Address{
			Role: ROLE_DINGODB_DOCUMENT,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoStoreRaftPort(),
		})
	case ROLE_DINGODB_DISKANN:
		address = append(address, Address{
			Role: ROLE_DINGODB_DISKANN,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoServerPort(),
		})
	case ROLE_DINGODB_INDEX:
		address = append(address, Address{
			Role: ROLE_DINGODB_INDEX,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoServerPort(),
		})
		address = append(address, Address{
			Role: ROLE_DINGODB_INDEX,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoStoreRaftPort(),
		})
	case ROLE_DINGODB_EXECUTOR:
		address = append(address, Address{
			Role: ROLE_DINGODB_EXECUTOR,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoDBServerPort(),
		})
		address = append(address, Address{
			Role: ROLE_DINGODB_EXECUTOR,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoDBMySQLPort(),
		})
	case ROLE_DINGODB_WEB:
		address = append(address, Address{
			Role: ROLE_DINGODB_WEB,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoDBServerPort(),
		})
		address = append(address, Address{
			Role: ROLE_DINGODB_WEB,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoDBExportPort(),
		})
	case ROLE_DINGODB_PROXY:
		address = append(address, Address{
			Role: ROLE_DINGODB_PROXY,
			IP:   dc.GetListenIp(),
			Port: dc.GetDingoDBServerPort(),
		})

	default:
		// do nothing
	}

	return address
}

func getServiceDirectorys(dc *topology.DeployConfig) []Directory {
	dirs := []Directory{}
	logDir := dc.GetLogDir()
	dataDir := dc.GetDataDir()
	sourceCoreDir := dc.GetSourceCoreDir()
	targetCoreDir := dc.GetTargetCoreDir()

	if len(logDir) > 0 {
		dirs = append(dirs, Directory{LOG_DIR, logDir})
	}
	if len(dataDir) > 0 {
		dirs = append(dirs, Directory{DATA_DIR, dataDir})
	}
	if len(sourceCoreDir) > 0 {
		dirs = append(dirs, Directory{CORE_DIR, sourceCoreDir})
	}
	if len(targetCoreDir) > 0 {
		dirs = append(dirs, Directory{CORE_DIR, targetCoreDir})
	}

	return dirs
}
