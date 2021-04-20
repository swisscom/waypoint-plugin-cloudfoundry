# waypoint-plugin-cloudfoundry

Plugin for waypoint that adds support for the cloudfoundry platform.

## Usage
Check out the project in the `example` folder to get an idea of how exactly to use the platform.

### Cloud Foundry deployment
```hcl
deploy {
   use "cloudfoundry" {
      organisation = "cf organisation"
      space = "waypoint-test"

      # App name can be overwritten, otherwise the application name from above is used.
      # app_name = "hello-world"

      # Make sure to create and rename this file, if needed
      # it should contain username:password as base64 encoded string
      docker_encoded_auth = file(abspath("./docker_encoded_credentials.secret"))
   }
}
```

### Cloud Foundry release
```hcl
release {
   use "cloudfoundry" {
      domain = "cfapp.swisscom.com"

      # Hostname can be specifically set, if it is different than the app name
      # hostname = my-example-app-url
   }
}
```

## Initial setup
### Mac OS
Install go:
`brew install go`

Install protoc-gen-go:
`go get google.golang.org/protobuf/cmd/protoc-gen-go`

Build:
`make`

Install locally (make sure the folder $HOME/.config/waypoint/plugins/ exists, if not create beforehand):
`make install`

## Building with Docker

To build plugins for release you can use the `build-docker` Makefile target, this will 
build your plugin for all architectures and create zipped artifacts which can be uploaded
to an artifact manager such as GitHub releases.

The built artifacts will be output in the `./releases` folder.

```shell
make build-docker

rm -rf ./releases
DOCKER_BUILDKIT=1 docker build --output releases --progress=plain .
#1 [internal] load .dockerignore
#1 transferring context: 2B done
#1 DONE 0.0s

#...

#14 [export_stage 1/1] COPY --from=build /go/plugin/bin/*.zip .
#14 DONE 0.1s

#15 exporting to client
#15 copying files 36.45MB 0.1s done
#15 DONE 0.1s
```

## Building and releasing with GitHub Actions

The action has two main phases:
1. **Build** - This phase builds the plugin binaries for all the supported architectures. It is triggered when pushing
   to a branch or on pull requests.
1. **Release** - This phase creates a new GitHub release containing the built plugin. It is triggered when pushing tags
   which starting with `v`, for example `v0.1.0`.
