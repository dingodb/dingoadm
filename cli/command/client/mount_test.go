package client

import (
	"testing"

	"github.com/dingodb/dingoadm/cli/cli"
)

func TestMountConfig(t *testing.T) {
	curveadm, err := cli.NewCurveAdm()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	var options = mountOptions{
		host:        "hostname",
		mountFSType: "s3",
		mountFSName: "test",
		mountPoint:  "{host_mount_point}",
		filename:    "{client_path}",
	}

	err = runMount(curveadm, options)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

}
