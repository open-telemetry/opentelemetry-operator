# Node.js auto-instrumentation

```bash
instrumentation.opentelemetry.io/inject-nodejs: "true"
```

Node.js resolves modules from the injected auto-instrumentation script path. If an instrumentation
package needs to load an application dependency from the application's `node_modules` directory, set
`NODE_PATH` with the application dependency path in the `Instrumentation` resource:

```yaml
spec:
  nodejs:
    env:
      - name: NODE_PATH
        value: /home/node/app/node_modules
```
