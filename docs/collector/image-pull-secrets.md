# Using imagePullSecrets

The OpenTelemetry Collector defines a ServiceAccount field which could be set to run collector instances with a specific Service and their properties (e.g. imagePullSecrets). Therefore, if you have a constraint to run your collector with a private container registry, you should follow the procedure below:

- Create Service Account.

```bash
kubectl create serviceaccount <service-account-name>
```

- Create an imagePullSecret.

```bash
kubectl create secret docker-registry <secret-name> --docker-server=<registry name> \
        --docker-username=DUMMY_USERNAME --docker-password=DUMMY_DOCKER_PASSWORD \
        --docker-email=DUMMY_DOCKER_EMAIL
```

- Add image pull secret to service account

```bash
kubectl patch serviceaccount <service-account-name> -p '{"imagePullSecrets": [{"name": "<secret-name>"}]}'
```
