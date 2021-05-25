package platform_test

import (
	"github.com/swisscom/waypoint-plugin-cloudfoundry/platform"
	"testing"
)

func TestGetServiceBindRepository(t *testing.T) {
	sbr, err := platform.GetServiceBindRepository()
	if err != nil {
		t.Fatalf("unable to get Service Bind repository")
	}

	var params map[string]interface{}
	err = sbr.Create(
		"cf1a8bf1-7948-4765-88a3-e278b00d8358",
		"e2d4d416-a6a0-4d73-b6fc-b5dff6c52f63",
		params,
	)

	if err != nil {
		t.Fatal(err)
	}
}
