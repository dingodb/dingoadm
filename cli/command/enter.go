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

package command

import (
	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/tools"
	"github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

type enterOptions struct {
	id string
}

func NewEnterCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options enterOptions

	cmd := &cobra.Command{
		Use:   "enter ID",
		Short: "Enter service container",
		Args:  utils.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			options.id = args[0]
			return dingoadm.CheckId(options.id)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnter(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	return cmd
}

func runEnter(dingoadm *cli.DingoAdm, options enterOptions) error {
	// 1) parse cluster topology
	dcs, err := dingoadm.ParseTopology()
	if err != nil {
		return err
	}

	// 2) filter service
	dcs = dingoadm.FilterDeployConfig(dcs, topology.FilterOption{
		Id:   options.id,
		Role: "*",
		Host: "*",
	})
	if len(dcs) == 0 {
		return errno.ERR_NO_SERVICES_MATCHED
	}

	// 3) get container id
	dc := dcs[0]
	serviceId := dingoadm.GetServiceId(dc.GetId())
	containerId, err := dingoadm.GetContainerId(serviceId)
	if err != nil {
		return err
	}

	// 4) attch remote container
	home := dc.GetProjectLayout().ServiceRootDir
	return tools.AttachRemoteContainer(dingoadm, dc.GetHost(), containerId, home)
}
