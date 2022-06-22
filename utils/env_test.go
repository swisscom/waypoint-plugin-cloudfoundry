package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/utils"
)

func TestParseEnv(t *testing.T) {
	envs := utils.ParseEnv("HELLO=world\n#comment\n#COMMENT=ok\nKUBECONFIG=none")
	assert.Equal(t, "world", envs["HELLO"])
	assert.Equal(t, "none", envs["KUBECONFIG"])
	assert.Len(t, envs, 2)
}
