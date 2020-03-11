# Root resources

Resources in this folder are applied once at cluster setup using a cluster
admin user. Only the definition of the `cadmin` account and its role bindings
exist here.

The `cadmin` account is used to continuously deploy resources to the `robocat`
cluster. If this account is modified, the associated secret in the `dogfooding`
cluster `robocat-tektoncd-cadmin-token` may have to be updated.
