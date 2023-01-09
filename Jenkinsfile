@Library('libpipelines') _

hose {
    EMAIL = 'eos'
    BUILDTOOL = 'make'
    DEVTIMEOUT = 30
    BUILDTOOL_IMAGE = 'golang:1.19.2'
    ANCHORE_POLICY = "production"
    VERSIONING_TYPE = 'stratioVersion-3-3'
    UPSTREAM_VERSION = '0.17.0'
    DEPLOYONPRS = true
    GRYPE_TEST = false

    DEV = { config ->
        doPackage(conf: config, parameters: "GOCACHE=/tmp")
        doDeploy(conf:config)
    }
}