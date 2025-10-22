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
 * Created Date: 2022-07-15
 * Author: Jingli Chen (Wine93)
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

package checker

import (
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/pkg/module"
)

type (
	step2CheckChunkfilePool struct {
		dc          *topology.DeployConfig
		execOptions module.ExecOptions
	}

	step2CheckS3 struct {
		s3AccessKey  string
		s3SecretKey  string
		s3Address    string
		s3BucketName string
	}

	step2CheckClientS3Configure struct {
		config *configure.ClientConfig
	}
)

func (s *step2CheckChunkfilePool) Execute(ctx *context.Context) error {
	dc := s.dc
	dataDir := dc.GetDataDir()
	if dc.GetEnableChunkfilePool() == false {
		return nil
	} else if len(dataDir) == 0 {
		return errno.ERR_CHUNKFILE_POOL_NOT_EXIST
	}

	var out string
	step := step.List{
		Files:       []string{dataDir},
		Out:         &out,
		ExecOptions: s.execOptions,
	}
	err := step.Execute(ctx)
	if err != nil {
		return err
	}

	// list file in data directory
	exist := map[string]bool{}
	files := strings.Split(out, "\n")
	for _, file := range files {
		exist[file] = true
	}

	// check wether the chunkfile pool exist
	files = []string{
		topology.LAYOUT_CURVEBS_CHUNKFILE_POOL_DIR,
		topology.METAFILE_CHUNKFILE_POOL,
	}
	for _, file := range files {
		if _, ok := exist[file]; !ok {
			return errno.ERR_CHUNKFILE_POOL_NOT_EXIST.
				F("%s (%s/%s: no such file or directory)",
					s.dc.GetHost(), dataDir, file)
		}
	}

	return nil
}

func (s *step2CheckS3) Execute(ctx *context.Context) error {
	/* TODO(P1): validate S3
	 * see also:
	 *	  https://aws.github.io/aws-sdk-go-v2/docs/getting-started/#to-get-your-access-key-id-and-secret-access-key
	 *	  https://www.programminghunter.com/article/7280107216/
	 */
	return nil
}

func (s *step2CheckClientS3Configure) Execute(ctx *context.Context) error {
	cc := s.config
	items := []struct {
		key   string
		value string
		err   *errno.ErrorCode
	}{
		{configure.KEY_CLIENT_S3_ACCESS_KEY, cc.GetS3AccessKey(), errno.ERR_INVALID_CURVEFS_CLIENT_S3_ACCESS_KEY},
		{configure.KEY_CLIENT_S3_SECRET_KEY, cc.GetS3SecretKey(), errno.ERR_INVALID_CURVEFS_CLIENT_S3_SECRET_KEY},
		{configure.KEY_CLIENT_S3_ADDRESS, cc.GetS3Address(), errno.ERR_INVALID_CURVEFS_CLIENT_S3_ADDRESS},
		{configure.KEY_CLIENT_S3_BUCKET_NAME, cc.GetS3BucketName(), errno.ERR_INVALID_CURVEFS_CLIENT_S3_BUCKET_NAME},
	}

	for _, item := range items {
		key := item.key
		value := item.value
		err := item.err
		if value == S3_TEMPLATE_VALUE || len(value) == 0 {
			return err.F("%s: %s", key, value)
		}
	}
	return nil
}

func NewCheckChunkfilePoolTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	hc, err := dingoadm.GetHost(dc.GetHost())
	if err != nil {
		return nil, err
	}

	subname := fmt.Sprintf("host=%s role=%s", dc.GetHost(), dc.GetRole())
	t := task.NewTask("Check Chunkfile Pool <service>", subname, hc.GetSSHConfig())

	t.AddStep(&step2CheckChunkfilePool{
		dc:          dc,
		execOptions: dingoadm.ExecOptions(),
	})

	return t, nil
}

func NewCheckS3Task(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	subname := fmt.Sprintf("host=%s role=%s", dc.GetHost(), dc.GetRole())
	t := task.NewTask("Check S3", subname, nil)

	t.AddStep(&step2CheckS3{
		s3AccessKey:  dc.GetS3AccessKey(),
		s3SecretKey:  dc.GetS3SecretKey(),
		s3Address:    dc.GetS3Address(),
		s3BucketName: dc.GetS3BucketName(),
	})

	return t, nil
}

func NewCheckMdsAddressTask(dingoadm *cli.DingoAdm, cc *configure.ClientConfig) (*task.Task, error) {
	host := dingoadm.MemStorage().Get(comm.KEY_CLIENT_HOST).(string)
	hc, err := dingoadm.GetHost(host)
	if err != nil {
		return nil, err
	}

	address := cc.GetClusterMDSAddr(dingoadm.MemStorage().Get(comm.KEY_FSTYPE).(string))
	subname := fmt.Sprintf("host=%s address=%s", host, address)
	t := task.NewTask("Check MDS Address", subname, hc.GetSSHConfig())

	return t, nil
}

func NewClientS3ConfigureTask(dingoadm *cli.DingoAdm, cc *configure.ClientConfig) (*task.Task, error) {
	t := task.NewTask("Check S3 Configure <service>", "", nil)

	t.AddStep(&step2CheckClientS3Configure{
		config: cc,
	})

	return t, nil
}

func NewCheckDiskUsageTask(dingoadm *cli.DingoAdm, cc *configure.ClientConfig) (*task.Task, error) {
	return nil, nil
}
