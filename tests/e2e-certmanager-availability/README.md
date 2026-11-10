# Cert-manager availability end-to-end test

Run this test only against an isolated cluster. It changes cert-manager permissions for the operator service account and waits for operator Deployments to roll out.

Install cert-manager and the operator with the current code. The operator service account must not already have cert-manager permissions. The controller Deployment and, if used, the separate webhook Deployment must each have at least two available replicas and use the same service account. The webhook Service must route to the webhook Deployment when it is separate.

Set `KUBECONFIG` to the test cluster's kubeconfig and run:

```sh
OTEL_CERTMANAGER_E2E=1 \
WEBHOOK_DEPLOYMENT=<webhook-deployment-name> \
go test -tags=e2e -count=1 -timeout=10m ./tests/e2e-certmanager-availability
```

Omit `WEBHOOK_DEPLOYMENT` when the webhook runs in the controller Deployment. `OPERATOR_NAMESPACE` and `OPERATOR_DEPLOYMENT` override the standard names. The test creates and removes its own namespace, ClusterRole, and ClusterRoleBinding. It fails if an admission request times out during a rollout, even when the Deployment reports available replicas.
