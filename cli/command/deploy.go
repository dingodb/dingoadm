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
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */
/*
* Project: dingoadm
* Author: dongwei (jackblack369)
 */

package command

import (
	"fmt"
	"time"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	utils "github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	CLEAN_PRECHECK_ENVIRONMENT = playbook.CLEAN_PRECHECK_ENVIRONMENT
	PULL_IMAGE                 = playbook.PULL_IMAGE
	CREATE_CONTAINER           = playbook.CREATE_CONTAINER
	CREATE_MDSV2_CLI_CONTAINER = playbook.CREATE_MDSV2_CLI_CONTAINER
	SYNC_CONFIG                = playbook.SYNC_CONFIG
	START_ETCD                 = playbook.START_ETCD
	ENABLE_ETCD_AUTH           = playbook.ENABLE_ETCD_AUTH
	START_MDS                  = playbook.START_MDS
	CREATE_PHYSICAL_POOL       = playbook.CREATE_PHYSICAL_POOL
	START_CHUNKSERVER          = playbook.START_CHUNKSERVER
	CREATE_LOGICAL_POOL        = playbook.CREATE_LOGICAL_POOL
	START_SNAPSHOTCLONE        = playbook.START_SNAPSHOTCLONE
	START_METASERVER           = playbook.START_METASERVER
	BALANCE_LEADER             = playbook.BALANCE_LEADER
	START_MDSV2                = playbook.START_MDS_V2
	START_COORDINATOR          = playbook.START_COORDINATOR
	START_STORE                = playbook.START_STORE
	START_MDSV2_CLI_CONTAINER  = playbook.START_MDSV2_CLI_CONTAINER
	START_DINGODB_EXECUTOR     = playbook.START_DINGODB_EXECUTOR
	SYNC_MDSV2_CONFIG          = playbook.SYNC_CONFIG
	CHECK_STORE_HEALTH         = playbook.CHECK_STORE_HEALTH
	CREATE_META_TABLES         = playbook.CREATE_META_TABLES
	SYNC_JAVA_OPTS             = playbook.SYNC_JAVA_OPTS

	// dingodb
	START_DINGODB_DOCUMENT = playbook.START_DINGODB_DOCUMENT
	START_DINGODB_INDEX    = playbook.START_DINGODB_INDEX
	START_DINGODB_DISKANN  = playbook.START_DINGODB_DISKANN
	START_DINGODB_PROXY    = playbook.START_DINGODB_PROXY
	START_DINGODB_WEB      = playbook.START_DINGODB_WEB

	// role
	ROLE_ETCD = topology.ROLE_ETCD
	// ROLE_MDS_V1           = topology.ROLE_MDS_V1
	ROLE_CHUNKSERVER      = topology.ROLE_CHUNKSERVER
	ROLE_SNAPSHOTCLONE    = topology.ROLE_SNAPSHOTCLONE
	ROLE_METASERVER       = topology.ROLE_METASERVER
	ROLE_MDS_V2           = topology.ROLE_MDS_V2
	ROLE_COORDINATOR      = topology.ROLE_COORDINATOR
	ROLE_STORE            = topology.ROLE_STORE
	ROLE_DINGODB_DOCUMENT = topology.ROLE_DINGODB_DOCUMENT
	ROLE_DINGODB_INDEX    = topology.ROLE_DINGODB_INDEX
	ROLE_DINGODB_DISKANN  = topology.ROLE_DINGODB_DISKANN
	ROLE_MDSV2_CLI        = topology.ROLE_MDSV2_CLI
	ROLE_DINGODB_EXECUTOR = topology.ROLE_DINGODB_EXECUTOR
	ROLE_DINGODB_WEB      = topology.ROLE_DINGODB_WEB
	ROLE_DINGODB_PROXY    = topology.ROLE_DINGODB_PROXY
)

var (
	CURVEBS_DEPLOY_STEPS = []int{
		CLEAN_PRECHECK_ENVIRONMENT,
		PULL_IMAGE,
		CREATE_CONTAINER,
		SYNC_CONFIG,
		START_ETCD,
		ENABLE_ETCD_AUTH,
		START_MDS,
		CREATE_PHYSICAL_POOL,
		START_CHUNKSERVER,
		CREATE_LOGICAL_POOL,
		START_SNAPSHOTCLONE,
		BALANCE_LEADER,
	}

	DINGOFS_DEPLOY_STEPS = []int{
		CLEAN_PRECHECK_ENVIRONMENT,
		PULL_IMAGE,
		CREATE_CONTAINER,
		SYNC_CONFIG,
		START_ETCD,
		ENABLE_ETCD_AUTH,
		START_MDS,
		CREATE_LOGICAL_POOL,
		START_METASERVER,
	}

	DINGOFS_MDS_DEPLOY_STEPS = []int{
		CLEAN_PRECHECK_ENVIRONMENT,
		PULL_IMAGE,
		CREATE_CONTAINER,
		SYNC_CONFIG,
		START_ETCD,
		ENABLE_ETCD_AUTH,
		START_MDS,
	}

	DINGOFS_MDSV2_ONLY_DEPLOY_STEPS = []int{
		CLEAN_PRECHECK_ENVIRONMENT,
		PULL_IMAGE,
		CREATE_CONTAINER,
		// SYNC_MDSV2_CONFIG
		// CREATE_META_TABLES,
		START_MDSV2,
	}

	DINGOFS_MDSV2_FOLLOW_DEPLOY_STEPS = []int{
		CLEAN_PRECHECK_ENVIRONMENT,
		PULL_IMAGE,
		CREATE_CONTAINER,
		CREATE_MDSV2_CLI_CONTAINER,
		SYNC_CONFIG,
		START_COORDINATOR,
		START_STORE,
		CHECK_STORE_HEALTH,
		START_MDSV2_CLI_CONTAINER,
		CREATE_META_TABLES,
		START_MDSV2,
		START_DINGODB_EXECUTOR,
	}

	DINGOSTORE_DEPLOY_STEPS = []int{
		CLEAN_PRECHECK_ENVIRONMENT,
		PULL_IMAGE,
		CREATE_CONTAINER,
		//SYNC_CONFIG,
		START_COORDINATOR,
		START_STORE,
	}

	DINGODB_DEPLOY_STEPS = []int{
		CLEAN_PRECHECK_ENVIRONMENT,
		PULL_IMAGE,
		CREATE_CONTAINER,
		SYNC_CONFIG,
		START_COORDINATOR,
		START_STORE,
		CHECK_STORE_HEALTH,
		START_DINGODB_DOCUMENT,
		START_DINGODB_DISKANN,
		START_DINGODB_INDEX,
		START_DINGODB_EXECUTOR,
		START_DINGODB_WEB,
		START_DINGODB_PROXY,
	}

	DEPLOY_FILTER_ROLE = map[int]string{
		START_ETCD:                 ROLE_ETCD,
		ENABLE_ETCD_AUTH:           ROLE_ETCD,
		START_MDS:                  ROLE_MDS_V2,
		START_CHUNKSERVER:          ROLE_CHUNKSERVER,
		START_SNAPSHOTCLONE:        ROLE_SNAPSHOTCLONE,
		START_METASERVER:           ROLE_METASERVER,
		CREATE_PHYSICAL_POOL:       ROLE_MDS_V2,
		CREATE_LOGICAL_POOL:        ROLE_MDS_V2,
		BALANCE_LEADER:             ROLE_MDS_V2,
		START_MDSV2:                ROLE_MDS_V2,
		START_COORDINATOR:          ROLE_COORDINATOR,
		START_STORE:                ROLE_STORE,
		START_DINGODB_DOCUMENT:     ROLE_DINGODB_DOCUMENT,
		START_DINGODB_DISKANN:      ROLE_DINGODB_DISKANN,
		START_DINGODB_INDEX:        ROLE_DINGODB_INDEX,
		START_MDSV2_CLI_CONTAINER:  ROLE_MDSV2_CLI,
		START_DINGODB_EXECUTOR:     ROLE_DINGODB_EXECUTOR,
		START_DINGODB_WEB:          ROLE_DINGODB_WEB,
		START_DINGODB_PROXY:        ROLE_DINGODB_PROXY,
		CHECK_STORE_HEALTH:         ROLE_STORE,
		CREATE_META_TABLES:         ROLE_MDSV2_CLI,
		CREATE_MDSV2_CLI_CONTAINER: ROLE_MDSV2_CLI,
		SYNC_JAVA_OPTS:             ROLE_DINGODB_EXECUTOR,
	}

	// DEPLOY_LIMIT_SERVICE is used to limit the number of services
	DEPLOY_LIMIT_SERVICE = map[int]int{
		CREATE_PHYSICAL_POOL:       1,
		CREATE_LOGICAL_POOL:        1,
		BALANCE_LEADER:             1,
		ENABLE_ETCD_AUTH:           1,
		CREATE_META_TABLES:         1,
		CREATE_MDSV2_CLI_CONTAINER: 1,
		CHECK_STORE_HEALTH:         1,
		SYNC_JAVA_OPTS:             1,
	}

	CAN_SKIP_ROLES = []string{
		ROLE_SNAPSHOTCLONE,
	}
)

type deployOptions struct {
	skip            []string
	insecure        bool
	poolset         string
	poolsetDiskType string
	useLocalImage   bool
}

func checkDeployOptions(options deployOptions) error {
	supported := utils.Slice2Map(CAN_SKIP_ROLES)
	for _, role := range options.skip {
		if !supported[role] {
			return errno.ERR_UNSUPPORT_SKIPPED_SERVICE_ROLE.
				F("skip role: %s", role)
		}
	}
	return nil
}

func NewDeployCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options deployOptions

	cmd := &cobra.Command{
		Use:   "deploy [OPTIONS]",
		Short: "Deploy cluster",
		Args:  cliutil.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkDeployOptions(options)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringSliceVar(&options.skip, "skip", []string{}, "Specify skipped service roles")
	flags.BoolVarP(&options.insecure, "insecure", "k", false, "Deploy without precheck")
	flags.StringVar(&options.poolset, "poolset", "default", "Specify the poolset name")
	flags.StringVar(&options.poolsetDiskType, "poolset-disktype", "ssd", "Specify the disk type of physical pool")
	flags.BoolVar(&options.useLocalImage, "local", false, "Use local image")

	return cmd
}

func skipServiceRole(deployConfigs []*topology.DeployConfig, options deployOptions) []*topology.DeployConfig {
	skipped := utils.Slice2Map(options.skip)
	dcs := []*topology.DeployConfig{}
	for _, dc := range deployConfigs {
		if skipped[dc.GetRole()] {
			continue
		}
		dcs = append(dcs, dc)
	}
	return dcs
}

func skipDeploySteps(dcs []*topology.DeployConfig, deploySteps []int, options deployOptions) []int {
	steps := []int{}
	skipped := utils.Slice2Map(options.skip)
	for _, step := range deploySteps {
		if (step == START_SNAPSHOTCLONE && skipped[ROLE_SNAPSHOTCLONE]) ||
			(step == ENABLE_ETCD_AUTH && len(dcs) > 0 && !dcs[0].GetEtcdAuthEnable()) {
			continue
		}
		steps = append(steps, step)
	}
	return steps
}

func precheckBeforeDeploy(dingoadm *cli.DingoAdm,
	dcs []*topology.DeployConfig,
	options deployOptions) error {
	// 1) skip precheck
	if options.insecure {
		return nil
	}

	// 2) generate precheck playbook
	pb, err := genPrecheckPlaybook(dingoadm, dcs, precheckOptions{
		skipSnapshotClone: utils.Slice2Map(options.skip)[ROLE_SNAPSHOTCLONE],
	})
	if err != nil {
		return err
	}

	// 3) run playbook
	err = pb.Run()
	if err != nil {
		return err
	}

	// 4) printf success prompt
	dingoadm.WriteOutln("")
	dingoadm.WriteOutln(color.GreenString("Congratulations!!! all precheck passed :)"))
	dingoadm.WriteOut(color.GreenString("Now we start to deploy cluster, sleep 3 seconds..."))
	time.Sleep(time.Duration(3) * time.Second)
	dingoadm.WriteOutln("\n")
	return nil
}

func calcNumOfChunkserver(dingoadm *cli.DingoAdm, dcs []*topology.DeployConfig) int {
	services := dingoadm.FilterDeployConfigByRole(dcs, topology.ROLE_CHUNKSERVER)
	return len(services)
}

func genDeployPlaybook(dingoadm *cli.DingoAdm,
	dcs []*topology.DeployConfig,
	options deployOptions) (*playbook.Playbook, error) {
	var steps []int
	kind := dcs[0].GetKind()

	// extract all deloy configs's role and deduplicate same role
	roles := dingoadm.GetRoles(dcs)

	switch kind {
	case topology.KIND_DINGOFS:
		if utils.Contains(roles, topology.ROLE_COORDINATOR) {
			if len(roles) == 1 {
				// only mds v2, no coordinator/store
				steps = DINGOFS_MDSV2_ONLY_DEPLOY_STEPS
			} else {
				// mds v2 with coordinator/store
				steps = DINGOFS_MDSV2_FOLLOW_DEPLOY_STEPS
				if !utils.Contains(roles, topology.ROLE_DINGODB_EXECUTOR) {
					// remove executor reference step which is the last step
					steps = steps[:len(steps)-1]
				}
			}
		} else if !utils.Contains(roles, topology.ROLE_METASERVER) {
			steps = DINGOFS_MDS_DEPLOY_STEPS
		} else {
			steps = DINGOFS_DEPLOY_STEPS
		}
	case topology.KIND_DINGOSTORE:
		steps = DINGOSTORE_DEPLOY_STEPS
	case topology.KIND_DINGODB:
		steps = DINGODB_DEPLOY_STEPS
	default:
		return nil, errno.ERR_UNSUPPORT_CLUSTER_KIND.F("kind: %s", kind)
	}

	if options.useLocalImage {
		// remove PULL_IMAGE step
		for i, item := range steps {
			if item == PULL_IMAGE {
				steps = append(steps[:i], steps[i+1:]...)
				break
			}
		}
	}
	steps = skipDeploySteps(dcs, steps, options) // not necessary
	poolset := configure.Poolset{
		Name: options.poolset,
		Type: options.poolsetDiskType,
	}
	diskType := options.poolsetDiskType

	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {
		// configs
		config := dcs
		if len(DEPLOY_FILTER_ROLE[step]) > 0 {
			role := DEPLOY_FILTER_ROLE[step]
			config = dingoadm.FilterDeployConfigByRole(config, role)
		}
		//n := len(config)
		if DEPLOY_LIMIT_SERVICE[step] > 0 {
			n := DEPLOY_LIMIT_SERVICE[step]
			config = config[:n]
		}

		// bs options
		options := map[string]interface{}{}
		if step == CREATE_PHYSICAL_POOL {
			options[comm.KEY_CREATE_POOL_TYPE] = comm.POOL_TYPE_PHYSICAL
			options[comm.KEY_POOLSET] = poolset
			options[comm.KEY_NUMBER_OF_CHUNKSERVER] = calcNumOfChunkserver(dingoadm, dcs)
		} else if step == CREATE_LOGICAL_POOL {
			options[comm.KEY_CREATE_POOL_TYPE] = comm.POOL_TYPE_LOGICAL
			options[comm.POOLSET] = poolset
			options[comm.POOLSET_DISK_TYPE] = diskType
			options[comm.KEY_NUMBER_OF_CHUNKSERVER] = calcNumOfChunkserver(dingoadm, dcs)
		}

		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: config,
			Options: options,
		})
	}
	return pb, nil
}

func statistics(dcs []*topology.DeployConfig) map[string]int {
	count := map[string]int{}
	for _, dc := range dcs {
		count[dc.GetRole()]++
	}
	return count
}

func serviceStats(dingoadm *cli.DingoAdm, dcs []*topology.DeployConfig) string {
	count := statistics(dcs)
	netcd := count[topology.ROLE_ETCD]
	nmds := count[topology.ROLE_MDS_V2]
	nchunkserevr := count[topology.ROLE_METASERVER]
	nsnapshotclone := count[topology.ROLE_SNAPSHOTCLONE]
	nmetaserver := count[topology.ROLE_METASERVER]

	var serviceStats string
	kind := dcs[0].GetKind()
	if kind == topology.KIND_CURVEBS { // KIND_CURVEBS
		serviceStats = fmt.Sprintf("etcd*%d, mds*%d, chunkserver*%d, snapshotclone*%d",
			netcd, nmds, nchunkserevr, nsnapshotclone)
	} else if kind == topology.KIND_DINGOFS {
		roles := dingoadm.GetRoles(dcs)
		if utils.Contains(roles, topology.ROLE_MDS_V2) {
			// mds v2
			ncoordinator := count[topology.ROLE_COORDINATOR]
			nstore := count[topology.ROLE_STORE]
			nmdsv2 := count[topology.ROLE_MDS_V2]
			nexecutor := count[topology.ROLE_DINGODB_EXECUTOR]
			serviceStats = fmt.Sprintf("coordinator*%d, store*%d, mds*%d, executor*%d", ncoordinator, nstore, nmdsv2, nexecutor)
		} else {
			// mds v1
			serviceStats = fmt.Sprintf("etcd*%d, mds*%d, metaserver*%d", netcd, nmds, nmetaserver)
		}
	} else if kind == topology.KIND_DINGOSTORE {
		ncoordinator := count[topology.ROLE_COORDINATOR]
		nstore := count[topology.ROLE_STORE]
		serviceStats = fmt.Sprintf("coordinator*%d, store*%d", ncoordinator, nstore)
	} else if kind == topology.KIND_DINGODB {
		ncoordinator := count[topology.ROLE_COORDINATOR]
		nstore := count[topology.ROLE_STORE]
		ndocument := count[topology.ROLE_DINGODB_DOCUMENT]
		ndiskann := count[topology.ROLE_DINGODB_DISKANN]
		nindex := count[topology.ROLE_DINGODB_INDEX]
		// nproxy := count[topology.ROLE_DINGODB_PROXY]
		// nweb := count[topology.ROLE_DINGODB_WEB]
		nexecutor := count[topology.ROLE_DINGODB_EXECUTOR]
		serviceStats = fmt.Sprintf("coordinator*%d, store*%d, document*%d, diskann*%d, index*%d, executor*%d",
			ncoordinator, nstore, ndocument, ndiskann, nindex, nexecutor)
	} else {
		serviceStats = "unknown"
	}

	return serviceStats
}

func displayDeployTitle(dingoadm *cli.DingoAdm, dcs []*topology.DeployConfig) {
	dingoadm.WriteOutln("Cluster Name    : %s", dingoadm.ClusterName())
	dingoadm.WriteOutln("Cluster Kind    : %s", dcs[0].GetKind())
	dingoadm.WriteOutln("Cluster Services: %s", serviceStats(dingoadm, dcs))
	dingoadm.WriteOutln("")
}

/*
 * Deploy Steps:
 *   1) pull image
 *   2) create container
 *   3) sync config
 *   4) start container
 *     4.1) start etcd container
 *     4.2) start mds container
 *     4.3) create physical pool(curvebs)
 *     4.3) start chunkserver(curvebs) / metaserver(dingofs) container
 *     4.4) start snapshotserver(curvebs) container
 *   5) create logical pool
 *   6) balance leader rapidly
 */
func runDeploy(dingoadm *cli.DingoAdm, options deployOptions) error {
	// 1) parse cluster topology
	dcs, err := dingoadm.ParseTopology()
	if err != nil {
		return err
	}

	// 2) skip service role
	dcs = skipServiceRole(dcs, options)

	// 3) precheck before deploy
	err = precheckBeforeDeploy(dingoadm, dcs, options)
	if err != nil {
		return err
	}

	// 4) generate deploy playbook
	pb, err := genDeployPlaybook(dingoadm, dcs, options)
	if err != nil {
		return err
	}

	// 5) display title
	displayDeployTitle(dingoadm, dcs)

	// 6) run playground
	if err = pb.Run(); err != nil {
		return err
	}

	// 7) print success prompt
	dingoadm.WriteOutln("")
	dingoadm.WriteOutln(color.GreenString("Cluster '%s' successfully deployed ^_^."), dingoadm.ClusterName())
	return nil
}
