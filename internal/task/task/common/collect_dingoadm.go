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
 * Created Date: 2022-08-14
 * Author: Jingli Chen (Wine93)
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

package common

import (
	"fmt"
	"path"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/internal/utils"
)

func NewCollectCurveAdmTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	// NOTE: we think it's not a good idae to collect curveadm's datbase file...
	// new task
	kind := dc.GetKind()
	subname := fmt.Sprintf("cluster=%s kind=%s",
		dingoadm.ClusterName(), kind)
	t := task.NewTask("Collect CurveAdm", subname, nil)

	// add step to task
	dbPath := dingoadm.Config().GetDBPath()
	secret := dingoadm.MemStorage().Get(comm.KEY_SECRET).(string)
	urlFormat := dingoadm.MemStorage().Get(comm.KEY_SUPPORT_UPLOAD_URL_FORMAT).(string)
	baseDir := dingoadm.TempDir()
	vname := utils.NewVariantName(fmt.Sprintf("curveadm_%s", utils.RandString(5)))
	localPath := path.Join(baseDir, vname.Name)                // /tmp/curveadm_is90x
	localTarballPath := path.Join(baseDir, vname.CompressName) // /tmp/curveadm_is90x.tar.gz
	localEncryptdTarballPath := path.Join(baseDir, vname.EncryptCompressName)
	httpSavePath := path.Join("/", encodeSecret(secret), "data")
	options := dingoadm.ExecOptions()
	options.ExecWithSudo = false
	options.ExecInLocal = true

	t.AddStep(&step.CreateDirectory{
		Paths:       []string{localPath /*, hostLogDir, hostConfDir*/},
		ExecOptions: options,
	})
	if len(dbPath) > 0 { // only copy local database (like sqlite)
		t.AddStep(&step.CopyFile{
			Source:      dbPath,
			Dest:        localPath,
			ExecOptions: options,
		})
	}
	t.AddStep(&step.Tar{
		File:        localPath,
		Archive:     localTarballPath,
		Create:      true,
		Gzip:        true,
		Verbose:     true,
		ExecOptions: options,
	})
	t.AddStep(&step2EncryptFile{
		source: localTarballPath,
		dest:   localEncryptdTarballPath,
		secret: secret,
	})
	t.AddStep(&step.Curl{ // upload to curve team
		Url:         fmt.Sprintf(urlFormat, httpSavePath),
		Form:        fmt.Sprintf("path=@%s", localEncryptdTarballPath),
		ExecOptions: options,
	})
	t.AddPostStep(&step.RemoveFile{
		Files:       []string{localPath, localTarballPath, localEncryptdTarballPath},
		ExecOptions: options,
	})

	return t, nil
}
