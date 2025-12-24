/*
 *  Copyright (c) 2021 NetEase Inc.
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
 * Created Date: 2021-11-25
 * Author: Jingli Chen (Wine93)
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 *
 * Project: Dingoadm
 * Author: Dongwei (jackblack369)
 */

package scripts

import (
	_ "embed"
)

var (
	// Common

	//go:embed shell/wait.sh
	WAIT string
	//go:embed shell/report.sh
	REPORT string

	// CurveBS

	//go:embed shell/format.sh
	FORMAT string
	//go:embed shell/wait_chunkserver.sh
	WAIT_CHUNKSERVERS string
	//go:embed shell/start_nginx.sh
	START_NGINX string
	//go:embed shell/create_volume.sh
	CREATE_VOLUME string
	//go:embed shell/map.sh
	MAP string
	//go:embed shell/target.sh
	TARGET string
	//go:embed shell/recycle.sh
	RECYCLE string

	// DingoFS

	//go:embed shell/mount_fs.sh
	MOUNT_FS string

	//go:embed shell/mount_client.sh
	MOUNT_CLIENT string

	//go:embed shell/start_gateway.sh
	START_GATEWAY string

	// DingoFS MdsV2
	//go:embed shell/create_mdsv2_tables.sh
	CREATE_MDSV2_TABLES string

	// DingoStore
	//go:embed shell/check_store_health.sh
	CHECK_STORE_HEALTH string

	// DingoFS Executor
	//go:embed shell/sync_java_opts.sh
	SYNC_JAVA_OPTS string

	// DingoFS Monitor
	//go:embed shell/start_monitor_sync.sh
	START_MONITOR_SYNC string

	// Prometheus Node Exporter
	//go:embed shell/sync_prometheus.sh
	SYNC_PROMETHEUS string

	// Grafana dashboard
	//go:embed shell/server_metric_zh.json
	GRAFANA_SERVER_METRIC string

	// Extract /etc/hosts mapping
	//go:embed shell/extract_hosts.sh
	EXTRACT_HOSTS string
)
