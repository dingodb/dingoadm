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
 * Created Date: 2022-07-24
 * Author: Jingli Chen (Wine93)
 */

// __SIGN_BY_WINE93__

package hosts

import (
	"github.com/dingodb/dingoadm/cli/cli"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

type showOptions struct{}

func NewShowCommand(curveadm *cli.DingoAdm) *cobra.Command {
	var options showOptions

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show hosts",
		Args:  cliutil.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(curveadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	return cmd
}

func runShow(curveadm *cli.DingoAdm, options showOptions) error {
	hosts := curveadm.Hosts()
	if len(hosts) == 0 {
		curveadm.WriteOutln("<empty hosts>")
	} else {
		curveadm.WriteOut(hosts)
	}
	return nil
}
