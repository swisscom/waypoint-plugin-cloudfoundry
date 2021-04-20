# The name of your project. A project typically maps 1:1 to a VCS repository.
# This name must be unique for your Waypoint server. If you're running in
# local mode, this must be unique to your machine.
project = "cloudfoundry-test"

# Labels can be specified for organizational purposes.
# labels = { "foo" = "bar" }

# An application to deploy.
app "hello-world" {
    # Build specifies how an application should be deployed. In this case,
    # we'll build using a Dockerfile and keeping it in a local registry.
    # For Cloud Foundry currently, only docker is supported.
    build {
        use "docker" {}
        
        registry {
            use "docker" {
                image = "waypoint-deploy-test"

                # Tags the image with the first 7 digits of the commit hash
                tag   = substr(gitrefhash(), 0, 7)
            }
        }

    }

    # Deploy to CF
    deploy {
        use "cloudfoundry" {
            organisation = "cf organisation"
            space = "waypoint-test"

            # App name can be overriden, else the application name from above is used.
            # app_name = "hello-world"

            # Make sure to create and rename this file, if needed
            docker_encoded_auth = file(abspath("./docker_encoded_credentials.secret"))
        }
    }

    # Release on CF
    release {
        use "cloudfoundry" {
            domain = "cfapp.swisscom.com"
        }
    }
}