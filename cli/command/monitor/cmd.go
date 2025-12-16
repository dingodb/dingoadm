/*
*  Copyright (c) 2023 NetEase Inc.
*  Copyright (c) 2025 dingodb.com.
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
* Project: Curveadm
* Created Date: 2023-04-17
* Author: wanghai (SeanHai)
*
* Project: dingoadm
* Author: dongwei (jackblack369)
 */

package monitor

import (
	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/cli/command/monitor/config"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

func NewMonitorCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Manage monitor",
		Args:  cliutil.NoArgs,
		RunE:  cliutil.ShowHelp(dingoadm.Err()),
	}

	cmd.AddCommand(
		NewDeployCommand(dingoadm),
		NewStartCommand(dingoadm),
		NewStopCommand(dingoadm),
		NewStatusCommand(dingoadm),
		NewCleanCommand(dingoadm),
		NewRestartCommand(dingoadm),
		NewReloadCommand(dingoadm),
		NewUpgradeCommand(dingoadm),
		config.NewConfigCommand(dingoadm),
	)
	return cmd
}
