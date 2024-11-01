package configure

import "testing"

func TestParseGatewayFile(t *testing.T) {
	fileName := "/home/dongw/dingofs/gateway.yaml"
	gc, err := ParseGatewayConfig(fileName)
	if err != nil {
		t.Errorf("parse gateway config file failed: %v", err)
	}
	t.Logf("gateway config: %v", gc)
}
