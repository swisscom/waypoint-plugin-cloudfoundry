package utils_test

import (
	proto "github.com/hashicorp/waypoint-plugin-sdk/proto/gen"
	"github.com/stretchr/testify/assert"
	"github.com/swisscom/waypoint-plugin-cloudfoundry/utils"
	"testing"
)

func TestHealthSummary(t *testing.T) {
	report1 := proto.StatusReport{
		Health:        proto.StatusReport_ALIVE,
		HealthMessage: "all processes are alive",
		External:      true,
	}

	report2 := proto.StatusReport{
		Health:        proto.StatusReport_DOWN,
		HealthMessage: "all processes are dead",
		External:      true,
	}

	assert.Equal(t,
		&proto.StatusReport{
			Resources: nil,
			Health: proto.StatusReport_DOWN,
			HealthMessage: "all processes are alive, all processes are dead",
			External: true,
		},
		utils.HealthSummary(&report1, &report2),
	)
}
