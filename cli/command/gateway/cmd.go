/*
 * Copyright (c) 2024 dingodb.com, Inc. All Rights Reserved
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * Project: DingoFS
 * Created Date: 2024-10-28
 * Author: Wei Dong (jackblack369)
 */

package gateway

import (
	"github.com/dingodb/curveadm/cli/cli"
	cliutil "github.com/dingodb/curveadm/internal/utils"
	"github.com/spf13/cobra"
)

func NewGatewayCommand(curveadm *cli.CurveAdm) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "Manage gateway",
		Args:  cliutil.NoArgs,
		RunE:  cliutil.ShowHelp(curveadm.Err()),
	}

	cmd.AddCommand(
		NewStartGatewayCommand(curveadm),
	)
	return cmd
}