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

package config

import (
	"errors"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

const (
	COMMIT_EXAMPLE = `Examples:
  $ dingoadm config commit /path/to/topology.yaml  # Commit cluster topology`
)

var (
	CHECK_TOPOLOGY_PLAYBOOK_STEPS = []int{
		playbook.CHECK_TOPOLOGY,
	}
)

type commitOptions struct {
	filename string
	slient   bool
	force    bool
}

func NewCommitCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options commitOptions

	cmd := &cobra.Command{
		Use:     "commit TOPOLOGY [OPTIONS]",
		Short:   "Commit cluster topology",
		Args:    utils.ExactArgs(1),
		Example: COMMIT_EXAMPLE,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.filename = args[0]
			return runCommit(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.slient, "slient", "s", false, "Slient output for config commit")
	flags.BoolVarP(&options.force, "force", "f", false, "Commit cluster topology by force")

	return cmd
}

func skipError(err error) bool {
	if errors.Is(err, errno.ERR_EMPTY_CLUSTER_TOPOLOGY) ||
		errors.Is(err, errno.ERR_NO_SERVICES_IN_TOPOLOGY) {
		return true
	}
	return false
}

func checkDiff(dingoadm *cli.DingoAdm, newData string) error {
	diffs, err := dingoadm.DiffTopology(dingoadm.ClusterTopologyData(), newData)
	if err != nil && !skipError(err) {
		return err
	}

	for _, diff := range diffs {
		dc := diff.DeployConfig
		switch diff.DiffType {
		case topology.DIFF_DELETE:
			return errno.ERR_DELETE_SERVICE_WHILE_COMMIT_TOPOLOGY_IS_DENIED.
				F("delete service: %s.host[%s]", dc.GetRole(), dc.GetHost())
		case topology.DIFF_ADD:
			return errno.ERR_ADD_SERVICE_WHILE_COMMIT_TOPOLOGY_IS_DENIED.
				F("added service: %s.host[%s]", dc.GetRole(), dc.GetHost())
		}
	}
	return nil
}

func genCheckTopologyPlaybook(dingoadm *cli.DingoAdm,
	dcs []*topology.DeployConfig,
	options commitOptions) (*playbook.Playbook, error) {
	steps := CHECK_TOPOLOGY_PLAYBOOK_STEPS
	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: nil,
			Options: map[string]interface{}{
				comm.KEY_ALL_DEPLOY_CONFIGS:       dcs,
				comm.KEY_CHECK_SKIP_SNAPSHOECLONE: false,
				comm.KEY_CHECK_WITH_WEAK:          true,
			},
			ExecOptions: playbook.ExecOptions{
				Concurrency:   100,
				SilentSubBar:  true,
				SilentMainBar: true,
			},
		})
	}
	return pb, nil
}

func readTopology(dingoadm *cli.DingoAdm, options commitOptions) (string, error) {
	filename := options.filename
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

	oldData := dingoadm.ClusterTopologyData()
	if !options.slient {
		diff := utils.Diff(oldData, data)
		dingoadm.WriteOutln("%s", diff)
	}
	return data, nil
}

func checkTopology(dingoadm *cli.DingoAdm, data string, options commitOptions) error {
	if options.force {
		return nil
	}

	// 1) check topology content is ok
	dcs, err := dingoadm.ParseTopologyData(data)
	if err != nil {
		return err
	}

	pb, err := genCheckTopologyPlaybook(dingoadm, dcs, options)
	if err != nil {
		return err
	}

	err = pb.Run()
	if err != nil {
		return err
	}

	// 2) check wether add/delete service
	if len(dingoadm.ClusterTopologyData()) > 0 {
		err = checkDiff(dingoadm, data)
		if err != nil {
			return err
		}
	}

	return nil
}

func runCommit(dingoadm *cli.DingoAdm, options commitOptions) error {
	// 1) parse cluster topology
	_, err := dingoadm.ParseTopology()
	if err != nil && !skipError(err) {
		return err
	}

	// 2) read  topology
	data, err := readTopology(dingoadm, options)
	if err != nil {
		return err
	}

	// 3) check topology
	err = checkTopology(dingoadm, data, options)
	if err != nil {
		return err
	}

	if !options.force {
		// 4) confirm by user
		if pass := tui.ConfirmYes("Do you want to continue?"); !pass {
			dingoadm.WriteOutln(tui.PromptCancelOpetation("commit topology"))
			return errno.ERR_CANCEL_OPERATION
		}
	}

	// 5) update cluster topology in database
	err = dingoadm.Storage().SetClusterTopology(dingoadm.ClusterId(), data)
	if err != nil {
		return errno.ERR_UPDATE_CLUSTER_TOPOLOGY_FAILED.E(err)
	}

	// 6) print success prompt
	dingoadm.WriteOutln("Cluster '%s' topology updated", dingoadm.ClusterName())
	return err
}
