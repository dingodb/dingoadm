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

package cluster

import (
	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	"github.com/dingodb/dingoadm/internal/utils"
	log "github.com/dingodb/dingoadm/pkg/log/glg"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	ADD_EXAMPLE = `Examples:
  $ dingoadm add my-cluster                            # Add a cluster named 'my-cluster'
  $ dingoadm add my-cluster -m "deploy for test"       # Add a cluster with description
  $ dingoadm add my-cluster -f /path/to/topology.yaml  # Add a cluster with specified topology`
)

var (
	CHECK_TOPOLOGY_PLAYBOOK_STEPS = []int{
		playbook.CHECK_TOPOLOGY,
	}
)

type addOptions struct {
	name        string
	descriotion string
	filename    string
	allowAbsent bool
}

func NewAddCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options addOptions

	cmd := &cobra.Command{
		Use:     "add CLUSTER [OPTIONS]",
		Short:   "Add cluster",
		Args:    utils.ExactArgs(1),
		Example: ADD_EXAMPLE,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.name = args[0]
			return runAdd(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.descriotion, "description", "m", "", "Description for cluster")
	flags.StringVarP(&options.filename, "topology", "f", "", "Specify the path of topology file")
	flags.BoolVarP(&options.allowAbsent, "absent", "", false, "Allow some service absent, default is false")

	return cmd
}

func readTopology(filename string) (string, error) {
	if len(filename) == 0 {
		return "", nil
	} else if !utils.PathExist(filename) {
		return "", errno.ERR_TOPOLOGY_FILE_NOT_FOUND.
			F("%s: no such file", utils.AbsPath(filename))
	}

	data, err := utils.ReadFile(filename)
	if err != nil {
		return "", errno.ERR_READ_TOPOLOGY_FILE_FAILED.E(err)
	}
	return data, nil
}

func genCheckTopologyPlaybook(dingoadm *cli.DingoAdm,
	dcs []*topology.DeployConfig,
	options addOptions) (*playbook.Playbook, error) {
	steps := CHECK_TOPOLOGY_PLAYBOOK_STEPS

	kind := dcs[0].GetKind()
	roles := dingoadm.GetRoles(dcs)

	if kind == topology.KIND_DINGOFS && !utils.Contains(roles, topology.ROLE_METASERVER) {
		// if no metaservers, allow absent metaservers
		options.allowAbsent = true
	}
	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: nil,
			Options: map[string]interface{}{
				comm.KEY_ALL_DEPLOY_CONFIGS:       dcs,
				comm.KEY_CHECK_SKIP_SNAPSHOECLONE: false,
				comm.KEY_CHECK_WITH_WEAK:          true,
				comm.KEY_ALLOW_ABSENT:             options.allowAbsent,
			},
			ExecOptions: playbook.ExecOptions{
				Concurrency:   100,
				SilentSubBar:  true,
				SilentMainBar: true,
				SkipError:     false,
			},
		})
	}
	return pb, nil
}

func checkTopology(dingoadm *cli.DingoAdm, data string, options addOptions) error {
	if len(options.filename) == 0 {
		return nil
	}

	dcs, err := dingoadm.ParseTopologyData(data)
	if err != nil {
		return err
	}

	pb, err := genCheckTopologyPlaybook(dingoadm, dcs, options)
	if err != nil {
		return err
	}
	return pb.Run()
}

func runAdd(dingoadm *cli.DingoAdm, options addOptions) error {
	// 1) check wether cluster already exist
	name := options.name
	storage := dingoadm.Storage()
	clusters, err := storage.GetClusters(name)
	if err != nil {
		log.Error("Get clusters failed",
			log.Field("cluster name", name),
			log.Field("error", err))
		return errno.ERR_GET_ALL_CLUSTERS_FAILED.E(err)
	} else if len(clusters) > 0 {
		return errno.ERR_CLUSTER_ALREADY_EXIST.
			F("cluster name: %s", name)
	}

	// 2) read topology iff specified and validte it
	data, err := readTopology(options.filename)
	if err != nil {
		return err
	}

	// 3) check topology
	err = checkTopology(dingoadm, data, options)
	if err != nil {
		return err
	}

	// 4) insert cluster (with topology) into database
	uuid := uuid.NewString()
	err = storage.InsertCluster(name, uuid, options.descriotion, data)
	if err != nil {
		return errno.ERR_INSERT_CLUSTER_FAILED.E(err)
	}

	// 5) print success prompt
	dingoadm.WriteOutln("Added cluster '%s'", name)
	return nil
}
