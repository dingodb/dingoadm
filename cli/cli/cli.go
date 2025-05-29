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

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	comm "github.com/dingodb/dingoadm/internal/common"
	configure "github.com/dingodb/dingoadm/internal/configure/dingoadm"
	"github.com/dingodb/dingoadm/internal/configure/hosts"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/storage"
	tools "github.com/dingodb/dingoadm/internal/tools/upgrade"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/internal/utils"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	log "github.com/dingodb/dingoadm/pkg/log/glg"
	"github.com/dingodb/dingoadm/pkg/module"
)

type DingoAdm struct {
	// project layout
	rootDir   string
	dataDir   string
	pluginDir string
	logDir    string
	tempDir   string
	logpath   string
	config    *configure.DingoAdmConfig

	// data pipeline
	in         io.Reader
	out        io.Writer
	err        io.Writer
	storage    *storage.Storage
	memStorage *utils.SafeMap

	// properties (hosts/cluster)
	hosts               string // hosts
	clusterId           int    // current cluster id
	clusterUUId         string // current cluster uuid
	clusterName         string // current cluster name
	clusterTopologyData string // cluster topology
	clusterPoolData     string // cluster pool
	monitor             storage.Monitor
}

/*
 * $HOME/.dingoadm
 *   - dingoadm.cfg
 *   - /bin/dingoadm
 *   - /data/dingoadm.db
 *   - /plugins/{shell,file,polarfs}
 *   - /logs/2006-01-02_15-04-05.log
 *   - /temp/
 */
func NewDingoAdm() (*DingoAdm, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errno.ERR_GET_USER_HOME_DIR_FAILED.E(err)
	}

	rootDir := fmt.Sprintf("%s/.dingoadm", home)
	dingoadm := &DingoAdm{
		rootDir:   rootDir,
		dataDir:   path.Join(rootDir, "data"),
		pluginDir: path.Join(rootDir, "plugins"),
		logDir:    path.Join(rootDir, "logs"),
		tempDir:   path.Join(rootDir, "temp"),
	}

	err = dingoadm.init()
	if err != nil {
		return nil, err
	}

	go dingoadm.detectVersion()
	return dingoadm, nil
}

func (dingoadm *DingoAdm) init() error {
	// (1) Create directory
	dirs := []string{
		dingoadm.rootDir,
		dingoadm.dataDir,
		dingoadm.pluginDir,
		dingoadm.logDir,
		dingoadm.tempDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return errno.ERR_CREATE_CURVEADM_SUBDIRECTORY_FAILED.E(err)
		}
	}

	// (2) Parse dingoadm.cfg
	confpath := fmt.Sprintf("%s/dingoadm.cfg", dingoadm.rootDir)
	config, err := configure.ParseDingoAdmConfig(confpath)
	if err != nil {
		return err
	}
	configure.ReplaceGlobals(config)

	// (3) Init logger
	now := time.Now().Format("2006-01-02_15-04-05")
	logpath := fmt.Sprintf("%s/dingoadm-%s.log", dingoadm.logDir, now)
	if err := log.Init(config.GetLogLevel(), logpath); err != nil {
		return errno.ERR_INIT_LOGGER_FAILED.E(err)
	} else {
		log.Info("Init logger success",
			log.Field("LogPath", logpath),
			log.Field("LogLevel", config.GetLogLevel()))
	}

	// (4) Init error code
	errno.Init(logpath)

	// (5) New storage: create table in sqlite/rqlite
	dbUrl := config.GetDBUrl()
	s, err := storage.NewStorage(dbUrl)
	if err != nil {
		log.Error("Init SQLite database failed",
			log.Field("Error", err))
		return errno.ERR_INIT_SQL_DATABASE_FAILED.E(err)
	}

	// (6) Get hosts
	var hosts storage.Hosts
	hostses, err := s.GetHostses()
	if err != nil {
		log.Error("Get hosts failed",
			log.Field("Error", err))
		return errno.ERR_GET_HOSTS_FAILED.E(err)
	} else if len(hostses) == 1 {
		hosts = hostses[0]
	}

	// (7) Get current cluster
	cluster, err := s.GetCurrentCluster()
	if err != nil {
		log.Error("Get current cluster failed",
			log.Field("Error", err))
		return errno.ERR_GET_CURRENT_CLUSTER_FAILED.E(err)
	} else {
		log.Info("Get current cluster success",
			log.Field("ClusterId", cluster.Id),
			log.Field("ClusterName", cluster.Name))
	}

	// (8) Get monitor configure
	monitor, err := s.GetMonitor(cluster.Id)
	if err != nil {
		log.Error("Get monitor failed", log.Field("Error", err))
		return errno.ERR_GET_MONITOR_FAILED.E(err)
	}

	dingoadm.logpath = logpath
	dingoadm.config = config
	dingoadm.in = os.Stdin
	dingoadm.out = os.Stdout
	dingoadm.err = os.Stderr
	dingoadm.storage = s
	dingoadm.memStorage = utils.NewSafeMap()
	dingoadm.hosts = hosts.Data
	dingoadm.clusterId = cluster.Id
	dingoadm.clusterUUId = cluster.UUId
	dingoadm.clusterName = cluster.Name
	dingoadm.clusterTopologyData = cluster.Topology
	dingoadm.clusterPoolData = cluster.Pool
	dingoadm.monitor = monitor

	return nil
}

func (dingoadm *DingoAdm) detectVersion() {
	latestVersion, err := tools.GetLatestVersion(Version)
	if err != nil || len(latestVersion) == 0 {
		return
	}

	versions, err := dingoadm.Storage().GetVersions()
	if err != nil {
		return
	} else if len(versions) > 0 {
		pendingVersion := versions[0].Version
		if pendingVersion == latestVersion {
			return
		}
	}

	dingoadm.Storage().SetVersion(latestVersion, "")
}

func (dingoadm *DingoAdm) Upgrade() (bool, error) {
	if dingoadm.config.GetAutoUpgrade() == false {
		return false, nil
	}

	versions, err := dingoadm.Storage().GetVersions()
	if err != nil || len(versions) == 0 {
		return false, nil
	}

	// (1) skip upgrade if the pending version is stale
	latestVersion := versions[0].Version
	err, yes := tools.IsLatest(Version, strings.TrimPrefix(latestVersion, "v"))
	if err != nil || yes {
		return false, nil
	}

	// (2) skip upgrade if user has confirmed
	day := time.Now().Format("2006-01-02")
	lastConfirm := versions[0].LastConfirm
	if day == lastConfirm {
		return false, nil
	}

	dingoadm.Storage().SetVersion(latestVersion, day)
	pass := tui.ConfirmYes(tui.PromptAutoUpgrade(latestVersion))
	if !pass {
		return false, errno.ERR_CANCEL_OPERATION
	}

	err = tools.Upgrade(latestVersion)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (dingoadm *DingoAdm) RootDir() string                   { return dingoadm.rootDir }
func (dingoadm *DingoAdm) DataDir() string                   { return dingoadm.dataDir }
func (dingoadm *DingoAdm) PluginDir() string                 { return dingoadm.pluginDir }
func (dingoadm *DingoAdm) LogDir() string                    { return dingoadm.logDir }
func (dingoadm *DingoAdm) TempDir() string                   { return dingoadm.tempDir }
func (dingoadm *DingoAdm) LogPath() string                   { return dingoadm.logpath }
func (dingoadm *DingoAdm) Config() *configure.DingoAdmConfig { return dingoadm.config }
func (dingoadm *DingoAdm) SudoAlias() string                 { return dingoadm.config.GetSudoAlias() }
func (dingoadm *DingoAdm) SSHTimeout() int                   { return dingoadm.config.GetSSHTimeout() }
func (dingoadm *DingoAdm) Engine() string                    { return dingoadm.config.GetEngine() }
func (dingoadm *DingoAdm) In() io.Reader                     { return dingoadm.in }
func (dingoadm *DingoAdm) Out() io.Writer                    { return dingoadm.out }
func (dingoadm *DingoAdm) Err() io.Writer                    { return dingoadm.err }
func (dingoadm *DingoAdm) Storage() *storage.Storage         { return dingoadm.storage }
func (dingoadm *DingoAdm) MemStorage() *utils.SafeMap        { return dingoadm.memStorage }
func (dingoadm *DingoAdm) Hosts() string                     { return dingoadm.hosts }
func (dingoadm *DingoAdm) ClusterId() int                    { return dingoadm.clusterId }
func (dingoadm *DingoAdm) ClusterUUId() string               { return dingoadm.clusterUUId }
func (dingoadm *DingoAdm) ClusterName() string               { return dingoadm.clusterName }
func (dingoadm *DingoAdm) ClusterTopologyData() string       { return dingoadm.clusterTopologyData }
func (dingoadm *DingoAdm) ClusterPoolData() string           { return dingoadm.clusterPoolData }
func (dingoadm *DingoAdm) Monitor() storage.Monitor          { return dingoadm.monitor }

func (dingoadm *DingoAdm) GetHost(host string) (*hosts.HostConfig, error) {
	if len(dingoadm.Hosts()) == 0 {
		return nil, errno.ERR_HOST_NOT_FOUND.
			F("host: %s", host)
	}
	hcs, err := hosts.ParseHosts(dingoadm.Hosts())
	if err != nil {
		return nil, err
	}

	for _, hc := range hcs {
		if hc.GetHost() == host {
			return hc, nil
		}
	}
	return nil, errno.ERR_HOST_NOT_FOUND.
		F("host: %s", host)
}

func (dingoadm *DingoAdm) ParseTopologyData(data string) ([]*topology.DeployConfig, error) {
	ctx := topology.NewContext()
	hcs, err := hosts.ParseHosts(dingoadm.Hosts())
	if err != nil {
		return nil, err
	}
	for _, hc := range hcs {
		ctx.Add(hc.GetHost(), hc.GetHostname())
	}

	dcs, err := topology.ParseTopology(data, ctx)
	if err != nil {
		return nil, err
	} else if len(dcs) == 0 {
		return nil, errno.ERR_NO_SERVICES_IN_TOPOLOGY
	}
	return dcs, err
}

func (dingoadm *DingoAdm) ParseTopology() ([]*topology.DeployConfig, error) {
	if dingoadm.ClusterId() == -1 {
		return nil, errno.ERR_NO_CLUSTER_SPECIFIED
	}
	return dingoadm.ParseTopologyData(dingoadm.ClusterTopologyData())
}

func (dingoadm *DingoAdm) FilterDeployConfig(deployConfigs []*topology.DeployConfig,
	options topology.FilterOption) []*topology.DeployConfig {
	dcs := []*topology.DeployConfig{}
	for _, dc := range deployConfigs {
		dcId := dc.GetId()
		role := dc.GetRole()
		host := dc.GetHost()
		serviceId := dingoadm.GetServiceId(dcId)
		if (options.Id == "*" || options.Id == serviceId) &&
			(options.Role == "*" || options.Role == role) &&
			(options.Host == "*" || options.Host == host) {
			dcs = append(dcs, dc)
		}
	}

	return dcs
}

func (dingoadm *DingoAdm) FilterDeployConfigByGateway(deployConfigs []*topology.DeployConfig,
	options topology.FilterOption) *topology.DeployConfig {
	for _, dc := range deployConfigs {
		host := dc.GetHost()
		if options.Host == host {
			return dc
		}
	}

	return nil
}

func (dingoadm *DingoAdm) FilterDeployConfigByRole(dcs []*topology.DeployConfig,
	role string) []*topology.DeployConfig {
	options := topology.FilterOption{Id: "*", Role: role, Host: "*"}
	return dingoadm.FilterDeployConfig(dcs, options)
}

func (dingoadm *DingoAdm) GetServiceId(dcId string) string {
	serviceId := fmt.Sprintf("%s_%s", dingoadm.ClusterUUId(), dcId)
	return utils.MD5Sum(serviceId)[:12]
}

func (dingoadm *DingoAdm) GetContainerId(serviceId string) (string, error) {
	containerId, err := dingoadm.Storage().GetContainerId(serviceId)
	if err != nil {
		return "", errno.ERR_GET_SERVICE_CONTAINER_ID_FAILED
	} else if len(containerId) == 0 {
		return "", errno.ERR_SERVICE_CONTAINER_ID_NOT_FOUND
	}
	return containerId, nil
}

// FIXME
func (dingoadm *DingoAdm) IsSkip(dc *topology.DeployConfig) bool {
	serviceId := dingoadm.GetServiceId(dc.GetId())
	containerId, err := dingoadm.Storage().GetContainerId(serviceId)
	return err == nil && len(containerId) == 0 && dc.GetRole() == topology.ROLE_SNAPSHOTCLONE
}

func (dingoadm *DingoAdm) GetVolumeId(host, user, volume string) string {
	volumeId := fmt.Sprintf("curvebs_volume_%s_%s_%s", host, user, volume)
	return utils.MD5Sum(volumeId)[:12]
}

func (dingoadm *DingoAdm) GetFilesystemId(host, mountPoint string) string {
	filesystemId := fmt.Sprintf("curvefs_filesystem_%s_%s", host, mountPoint)
	return utils.MD5Sum(filesystemId)[:12]
}

func (dingoadm *DingoAdm) ExecOptions() module.ExecOptions {
	return module.ExecOptions{
		ExecWithSudo:   true,
		ExecInLocal:    false,
		ExecSudoAlias:  dingoadm.config.GetSudoAlias(),
		ExecTimeoutSec: dingoadm.config.GetTimeout(),
		ExecWithEngine: dingoadm.config.GetEngine(),
	}
}

func (dingoadm *DingoAdm) CheckId(id string) error {
	services, err := dingoadm.Storage().GetServices(dingoadm.ClusterId())
	if err != nil {
		return err
	}
	for _, service := range services {
		if service.Id == id {
			return nil
		}
	}
	return errno.ERR_ID_NOT_FOUND.F("id: %s", id)
}

func (dingoadm *DingoAdm) CheckRole(role string) error {
	dcs, err := dingoadm.ParseTopology()
	if err != nil {
		return err
	}

	kind := dcs[0].GetKind()
	roles := topology.CURVEBS_ROLES
	if kind == topology.KIND_CURVEFS || kind == topology.KIND_DINGOFS {
		roles = topology.DINGOFS_ROLES
	}
	supported := utils.Slice2Map(roles)
	if !supported[role] {
		if kind == topology.KIND_CURVEBS {
			return errno.ERR_UNSUPPORT_CURVEBS_ROLE.
				F("role: %s", role)
		}
		return errno.ERR_UNSUPPORT_CURVEFS_ROLE.
			F("role: %s", role)
	}
	return nil
}

func (dingoadm *DingoAdm) CheckHost(host string) error {
	_, err := dingoadm.GetHost(host)
	return err
}

// writer for cobra command error
func (dingoadm *DingoAdm) Write(p []byte) (int, error) {
	// trim prefix which generate by cobra
	p = p[len(cliutil.PREFIX_COBRA_COMMAND_ERROR):]
	return dingoadm.WriteOut(string(p))
}

func (dingoadm *DingoAdm) WriteOut(format string, a ...interface{}) (int, error) {
	output := fmt.Sprintf(format, a...)
	return dingoadm.out.Write([]byte(output))
}

func (dingoadm *DingoAdm) WriteOutln(format string, a ...interface{}) (int, error) {
	output := fmt.Sprintf(format, a...) + "\n"
	return dingoadm.out.Write([]byte(output))
}

func (dingoadm *DingoAdm) IsSameRole(dcs []*topology.DeployConfig) bool {
	role := dcs[0].GetRole()
	for _, dc := range dcs {
		if dc.GetRole() != role {
			return false
		}
	}
	return true
}

func (dingoadm *DingoAdm) DiffTopology(data1, data2 string) ([]topology.TopologyDiff, error) {
	ctx := topology.NewContext()
	hcs, err := hosts.ParseHosts(dingoadm.Hosts())
	if err != nil {
		return nil, err
	}
	for _, hc := range hcs {
		ctx.Add(hc.GetHost(), hc.GetHostname())
	}

	if len(data1) == 0 {
		return nil, errno.ERR_EMPTY_CLUSTER_TOPOLOGY
	}

	dcs, err := topology.ParseTopology(data1, ctx)
	if err != nil {
		return nil, err // err is error code
	}
	if len(dcs) == 0 {
		return nil, errno.ERR_NO_SERVICES_IN_TOPOLOGY
	}
	return topology.DiffTopology(data1, data2, ctx)
}

func (dingoadm *DingoAdm) PreAudit(now time.Time, args []string) int64 {
	if len(args) == 0 {
		return -1
	} else if args[0] == "audit" || args[0] == "__complete" {
		return -1
	}

	cwd, _ := os.Getwd()
	command := fmt.Sprintf("dingoadm %s", strings.Join(args, " "))
	id, err := dingoadm.Storage().InsertAuditLog(
		now, cwd, command, comm.AUDIT_STATUS_ABORT)
	if err != nil {
		log.Error("Insert audit log failed",
			log.Field("Error", err))
	}

	return id
}

func (dingoadm *DingoAdm) PostAudit(id int64, ec error) {
	if id < 0 {
		return
	}

	auditLogs, err := dingoadm.Storage().GetAuditLog(id)
	if err != nil {
		log.Error("Get audit log failed",
			log.Field("Error", err))
		return
	} else if len(auditLogs) != 1 {
		return
	}

	auditLog := auditLogs[0]
	status := auditLog.Status
	errorCode := 0
	if ec == nil {
		status = comm.AUDIT_STATUS_SUCCESS
	} else if errors.Is(ec, errno.ERR_CANCEL_OPERATION) {
		status = comm.AUDIT_STATUS_CANCEL
	} else {
		status = comm.AUDIT_STATUS_FAIL
		if v, ok := ec.(*errno.ErrorCode); ok {
			errorCode = v.GetCode()
		}
	}

	err = dingoadm.Storage().SetAuditLogStatus(id, status, errorCode)
	if err != nil {
		log.Error("Set audit log status failed",
			log.Field("Error", err))
	}
}

func (dingoadm *DingoAdm) SwitchCluster(cluster storage.Cluster) error {

	dingoadm.memStorage = utils.NewSafeMap()
	dingoadm.clusterId = cluster.Id
	dingoadm.clusterUUId = cluster.UUId
	dingoadm.clusterName = cluster.Name
	dingoadm.clusterTopologyData = cluster.Topology
	dingoadm.clusterPoolData = cluster.Pool

	return nil
}
