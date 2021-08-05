module github.com/swisscom/waypoint-plugin-cloudfoundry

go 1.14

require (
	code.cloudfoundry.org/cli v0.0.0-20201210201943-be4a5ce2b96e
	github.com/google/uuid v1.1.2
	github.com/hashicorp/go-hclog v0.16.1
	github.com/hashicorp/waypoint v0.3.2
	github.com/hashicorp/waypoint-plugin-sdk v0.0.0-20210714195507-d64bfe0a3ffa
	github.com/stretchr/testify v1.6.1
	google.golang.org/protobuf v1.26.0
	k8s.io/apimachinery v0.19.4
)

// Pin because of https://github.com/cloudfoundry/log-cache-release/issues/41
replace google.golang.org/protobuf => google.golang.org/protobuf v1.25.0

replace code.cloudfoundry.org/cli => github.com/swisscom/cloudfoundry-cli v0.0.0-20210521140856-c01ddabd05f7
