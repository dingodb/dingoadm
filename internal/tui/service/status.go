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
 * Created Date: 2021-10-15
 * Author: Jingli Chen (Wine93)
 */

// __SIGN_BY_WINE93__

package service

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	task "github.com/dingodb/dingoadm/internal/task/task/common"
	"github.com/dingodb/dingoadm/internal/task/task/monitor"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	longest "github.com/jpillora/longestcommon"
)

const (
	ROLE_ETCD          = topology.ROLE_ETCD
	ROLE_MDS           = topology.ROLE_MDS
	ROLE_CHUNKSERVER   = topology.ROLE_CHUNKSERVER
	ROLE_METASERVER    = topology.ROLE_METASERVER
	ROLE_SNAPSHOTCLONE = topology.ROLE_SNAPSHOTCLONE
	ROLE_COORDINATOR   = topology.ROLE_COORDINATOR
	ROLE_STORE         = topology.ROLE_STORE

	ITEM_ID = iota
	ITEM_CONTAINER_ID
	ITEM_STATUS
	ITEM_PORTS
	ITEM_LOG_DIR
	ITEM_DATA_DIR
	ITEM_RAFT_DIR

	STATUS_CLEANED = comm.SERVICE_STATUS_CLEANED
	STATUS_LOSED   = comm.SERVICE_STATUS_LOSED
	STATUS_UNKNWON = comm.SERVICE_STATUS_UNKNOWN
	// for instance merged status
	STATUS_RUNNING  = "RUNNING"
	STATUS_STOPPED  = "STOPPED"
	STATUS_ABNORMAL = "ABNORMAL"
)

var (
	ROLE_SCORE = map[string]int{
		ROLE_ETCD:          0,
		ROLE_COORDINATOR:   0,
		ROLE_MDS:           1,
		ROLE_STORE:         1,
		ROLE_CHUNKSERVER:   2,
		ROLE_METASERVER:    2,
		ROLE_SNAPSHOTCLONE: 3,
	}
	MONITOT_ROLE_SCORE = map[string]int{
		configure.ROLE_NODE_EXPORTER: 0,
		configure.ROLE_PROMETHEUS:    1,
		configure.ROLE_GRAFANA:       2,
	}
)

func statusDecorate(status string) string {
	switch status {
	case STATUS_CLEANED:
		return color.BlueString(status)
	case STATUS_LOSED, STATUS_UNKNWON, STATUS_ABNORMAL:
		return color.RedString(status)
	}
	return status
}

func sortStatues(statuses []task.ServiceStatus) {
	sort.Slice(statuses, func(i, j int) bool {
		s1, s2 := statuses[i], statuses[j]
		c1, c2 := s1.Config, s2.Config
		if s1.Role == s2.Role {
			if c1.GetHostSequence() == c2.GetHostSequence() {
				return c1.GetInstancesSequence() < c2.GetInstancesSequence()
			}
			return c1.GetHostSequence() < c2.GetHostSequence()
		}
		return ROLE_SCORE[s1.Role] < ROLE_SCORE[s2.Role]
	})
}

func id(items []string) string {
	if len(items) == 1 {
		return items[0]
	}
	return "<instances>"
}

func status(items []string) string {
	if len(items) == 1 {
		return items[0]
	}

	count := map[string]int{}
	for _, item := range items {
		status := item
		if strings.HasPrefix(item, "Up") {
			status = STATUS_RUNNING
		} else if strings.HasPrefix(item, "Exited") {
			status = STATUS_STOPPED
		}
		count[status]++
	}

	for status, n := range count {
		if n == len(items) {
			return status
		}
	}
	return STATUS_ABNORMAL
}

func dir(items []string) string {
	if len(items) == 1 {
		return items[0]
	}

	prefix := longest.Prefix(items)
	first := strings.TrimPrefix(items[0], prefix)
	last := strings.TrimPrefix(items[len(items)-1], prefix)
	limit := utils.Min(5, len(first), len(last))
	return fmt.Sprintf("%s{%s...%s}", prefix, first[:limit], last[:limit])
}

func merge(statuses []task.ServiceStatus, item int) string {
	items := []string{}
	for _, status := range statuses {
		switch item {
		case ITEM_ID:
			items = append(items, status.Id)
		case ITEM_CONTAINER_ID:
			items = append(items, status.ContainerId)
		case ITEM_STATUS:
			items = append(items, status.Status)
		case ITEM_PORTS:
			items = append(items, status.Ports)
		case ITEM_LOG_DIR:
			items = append(items, status.LogDir)
		case ITEM_DATA_DIR:
			items = append(items, status.DataDir)
		case ITEM_RAFT_DIR:
			items = append(items, status.RaftDir)
		}
	}

	sort.Strings(items)
	switch item {
	case ITEM_ID:
		return id(items)
	case ITEM_CONTAINER_ID:
		return id(items)
	case ITEM_STATUS:
		return status(items)
	case ITEM_PORTS:
		return id(items)
	case ITEM_LOG_DIR:
		return dir(items)
	case ITEM_DATA_DIR:
		return dir(items)
	case ITEM_RAFT_DIR:
		return dir(items)
	}
	return ""
}

func mergeStatues(statuses []task.ServiceStatus) []task.ServiceStatus {
	ss := []task.ServiceStatus{}
	i, j, n := 0, 0, len(statuses)
	for i = 0; i < n; i++ {
		for j = i + 1; j < n && statuses[i].ParentId == statuses[j].ParentId; j++ {
		}
		status := statuses[i]
		ss = append(ss, task.ServiceStatus{
			Id:          merge(statuses[i:j], ITEM_ID),
			Role:        status.Role,
			Host:        status.Host,
			Instances:   fmt.Sprintf("%d/%s", j-i, strings.Split(status.Instances, "/")[1]),
			ContainerId: merge(statuses[i:j], ITEM_CONTAINER_ID),
			Status:      merge(statuses[i:j], ITEM_STATUS),
			Ports:       merge(statuses[i:j], ITEM_PORTS),
			LogDir:      merge(statuses[i:j], ITEM_LOG_DIR),
			DataDir:     merge(statuses[i:j], ITEM_DATA_DIR),
			RaftDir:     merge(statuses[i:j], ITEM_RAFT_DIR),
		})
		i = j - 1
	}
	return ss
}

func FormatStatus(kind string, statuses []task.ServiceStatus, verbose, expand bool, excludeCols []string) (string, int) {
	lines := [][]interface{}{}

	// title
	title := []string{
		"Id",
		"Role",
		"Host",
		"Instances",
		"Container Id",
		"Status",
		"Ports",
		"Log Dir",
		"Data Dir",
	}

	if len(excludeCols) != 0 {
		// Create exclusion set
		excludeSet := make(map[string]struct{}, len(excludeCols))
		for _, ex := range excludeCols {
			excludeSet[ex] = struct{}{}
		}

		// Remove excluded items using efficient slice mutation
		title = slices.DeleteFunc(title, func(t string) bool {
			_, exists := excludeSet[t]
			return exists
		})
	}

	if kind == topology.KIND_DINGOSTORE {
		title = append(title, "Raft Dir")
	}

	first, second := tui.FormatTitle(title)
	lines = append(lines, first)
	lines = append(lines, second)

	// status
	sortStatues(statuses)
	if !expand {
		statuses = mergeStatues(statuses)
	}
	for _, status := range statuses {
		if kind == topology.KIND_DINGOSTORE {
			lines = append(lines, []interface{}{
				status.Id,
				status.Role,
				status.Host,
				status.Instances,
				status.ContainerId,
				tui.DecorateMessage{Message: status.Status, Decorate: statusDecorate},
				utils.Choose(len(status.Ports) == 0, "-", status.Ports),
				status.LogDir,
				status.DataDir,
				status.RaftDir,
			})
		} else {
			lines = append(lines, []interface{}{
				status.Id,
				status.Role,
				status.Host,
				status.Instances,
				status.ContainerId,
				tui.DecorateMessage{Message: status.Status, Decorate: statusDecorate},
				utils.Choose(len(status.Ports) == 0, "-", status.Ports),
				status.LogDir,
				status.DataDir,
			})
		}
	}

	// cut column
	locate := utils.Locate(title)
	if !verbose {
		tui.CutColumn(lines, locate["Ports"])    // Ports info
		tui.CutColumn(lines, locate["Data Dir"]) // Data Dir
		tui.CutColumn(lines, locate["Log Dir"])  // Log Dir
		tui.CutColumn(lines, locate["Raft Dir"]) // Raft Dir
	}

	output := tui.FixedFormat(lines, 2)
	lastLine := fmt.Sprint(lines[len(lines)-1]...)
	width := len(lastLine) + len(title)*2
	return output, width
}

func sortMonitorStatues(statuses []monitor.MonitorStatus) {
	sort.Slice(statuses, func(i, j int) bool {
		s1, s2 := statuses[i], statuses[j]
		if s1.Role == s2.Role {
			return s1.Host < s1.Host
		}
		return MONITOT_ROLE_SCORE[s1.Role] < ROLE_SCORE[s2.Role]
	})
}

func FormatMonitorStatus(statuses []monitor.MonitorStatus, verbose bool) string {
	lines := [][]interface{}{}

	// title
	title := []string{
		"Id",
		"Role",
		"Host",
		"Container Id",
		"Status",
		"Ports",
		"Data Dir",
	}
	first, second := tui.FormatTitle(title)
	lines = append(lines, first)
	lines = append(lines, second)

	// status
	sortMonitorStatues(statuses)
	for _, status := range statuses {
		lines = append(lines, []interface{}{
			status.Id,
			status.Role,
			status.Host,
			status.ContainerId,
			tui.DecorateMessage{Message: status.Status, Decorate: statusDecorate},
			utils.Choose(len(status.Ports) == 0, "-", status.Ports),
			status.DataDir,
		})
	}

	// cut column
	locate := utils.Locate(title)
	if !verbose {
		tui.CutColumn(lines, locate["Ports"])    // Data Dir
		tui.CutColumn(lines, locate["Data Dir"]) // Data Dir
	}

	output := tui.FixedFormat(lines, 2)
	return output
}
