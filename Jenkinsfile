@Library('libpipelines') _

hose {
    EMAIL = 'eos'
    BUILDTOOL = 'make'
    DEVTIMEOUT = 30
    BUILDTOOL_IMAGE = 'golang:1.19.8'
    ANCHORE_POLICY = "production"
    VERSIONING_TYPE = 'stratioVersion-3-3'
    UPSTREAM_VERSION = '0.17.0'
    DEPLOYONPRS = true
    GRYPE_TEST = false
    MODULE_LIST = [ "paas.cloud-provisioner:cloud-provisioner:tar.gz"]

    DEV = { config ->
        doPackage(conf: config, parameters: "GOCACHE=/tmp")
        doDeploy(conf: config)
    }
    BUILDTOOL_MEMORY_REQUEST = "512Mi"
    BUILDTOOL_MEMORY_LIMIT = "2048Mi"
}
