/*
 * 	Copyright (c) 2025 dingodb.com Inc.
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

package cluster

import (
	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/errno"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	log "github.com/dingodb/dingoadm/pkg/log/glg"
	"github.com/spf13/cobra"
)

type renameOptions struct {
	clusterOldName string
	clusterNewName string
	force          bool
}

func NewRenameCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options renameOptions

	cmd := &cobra.Command{
		Use:   "rename CLUSTER [OPTIONS]",
		Short: "Rename cluster",
		Args:  cliutil.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.clusterOldName = args[0]
			options.clusterNewName = args[1]

			return runRename(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.BoolVarP(&options.force, "force", "f", false, "Remove cluster by force")

	return cmd
}

func runRename(dingoadm *cli.DingoAdm, options renameOptions) error {
	// 1) get cluster by name
	storage := dingoadm.Storage()
	clusterOldName := options.clusterOldName
	clusterNewName := options.clusterNewName
	clusters, err := storage.GetClusters(clusterOldName) // Get all clusters
	if err != nil {
		log.Error("Get cluster failed",
			log.Field("error", err))
		return errno.ERR_GET_ALL_CLUSTERS_FAILED.E(err)
	} else if len(clusters) == 0 {
		return errno.ERR_CLUSTER_NOT_FOUND.
			F("cluster name: %s", clusterOldName)
	}

	// rename cluster name
	if options.force {
		dingoadm.WriteOut(tui.PromptRenameCluster(clusterOldName, clusterNewName))
	} else {
		if !tui.ConfirmYes(tui.PromptRenameCluster(clusterOldName, clusterNewName)) {
			dingoadm.WriteOut(tui.PromptCancelOpetation("rename cluster"))
			return errno.ERR_CANCEL_OPERATION
		}
	}

	if err := dingoadm.Storage().RenameClusterName(clusterOldName, clusterNewName); err != nil {
		return errno.ERR_RENAME_CLUSTER_FAILED.E(err)
	}

	// 3) print success prompt
	dingoadm.WriteOutln("Rename cluster '%s' to '%s'", clusterOldName, clusterNewName)
	return nil
}
