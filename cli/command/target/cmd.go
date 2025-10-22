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
 * Created Date: 2022-02-08
 * Author: Jingli Chen (Wine93)
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

package target

import (
	"github.com/dingodb/dingoadm/cli/cli"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

func NewTargetCommand(curveadm *cli.DingoAdm) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "target",
		Short: "Manage SCSI target of CurveBS",
		Args:  cliutil.NoArgs,
		RunE:  cliutil.ShowHelp(curveadm.Err()),
	}

	cmd.AddCommand(
		NewStartCommand(curveadm),
		NewStopCommand(curveadm),
		NewAddCommand(curveadm),
		NewDeleteCommand(curveadm),
		NewListCommand(curveadm),
	)
	return cmd
}
