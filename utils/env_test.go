package utils_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/utils"
	"testing"
)

func TestParseEnv(t *testing.T){
	envs := utils.ParseEnv("HELLO=world\n#comment\n#COMMENT=ok\nKUBECONFIG=none")
	assert.Equal(t, "world", envs["HELLO"])
	assert.Equal(t, "none", envs["KUBECONFIG"])
	assert.Len(t, envs, 2)
}
