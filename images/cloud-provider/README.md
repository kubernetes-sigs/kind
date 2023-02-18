# Cloud Provider Kind

With the removal of the in-tree cloud providers, there is a gap in testing, mainly
in the next two areas:

- Node lifecycle
- Load Balancers

This is an implementation of a external cloud provider for KIND that it implements thee
external Cloud Provider interaface and acts as a Cloud Provider for the KIND cluster.

References:

- https://github.com/kubernetes/cloud-provider
- https://github.com/kubernetes/enhancements/tree/master/keps/sig-cloud-provider/2395-removing-in-tree-cloud-providers