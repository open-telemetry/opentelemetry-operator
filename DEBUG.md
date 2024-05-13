# Debug tips to the OpenTelemetry Operator

A tip during a troubleshooting process is always welcome. Therefore, we prepared this documentation to help you identify possible issues and improve the application's reliability.

## Customizing Logs Output
By the default the Operator's log format is console like you can see below:
```bash
2024-05-06T11:55:11+02:00	INFO	setup	Prometheus CRDs are installed, adding to scheme.
2024-05-06T11:55:11+02:00	INFO	setup	Openshift CRDs are not installed, skipping adding to scheme.
2024-05-06T11:55:11+02:00	INFO	setup	the env var WATCH_NAMESPACE isn't set, watching all namespaces
2024-05-06T11:55:11+02:00	INFO	Webhooks are disabled, operator is running an unsupported mode	{"ENABLE_WEBHOOKS": "false"}
2024-05-06T11:55:11+02:00	INFO	setup	starting manager
```

If it is necessary to customize the log format, so you can use one of the following parameters:
- `--zap-devel`:                                        Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) (default false)
- `--zap-encoder`:                               Zap log encoding (one of 'json' or 'console')
- `--zap-log-level`                              Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
- `--zap-stacktrace-level`                        Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
- `--zap-time-encoding`                   Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.
- The following parameters are effective only if the `--zap-encoder=json`:
  - `zap-message-key`: The message key to be used in the customized Log Encoder
  - `zap-level-key`: The level key to be used in the customized Log Encoder
  - `zap-time-key`: The time key to be used in the customized Log Encoder
  - `zap-level-format`: The level format to be used in the customized Log Encoder

Running the Operator with the parameters `--zap-encoder=json`, `--zap-message-key="msg"`, `zap-level-key="severity"`,`zap-time-key="timestamp"`,`zap-level-format="uppercase"` you should see the following output:
```bash
{"severity":"INFO","timestamp":"2024-05-07T16:23:35+02:00","logger":"setup","msg":"Prometheus CRDs are installed, adding to scheme."}
{"severity":"INFO","timestamp":"2024-05-07T16:23:35+02:00","logger":"setup","msg":"Openshift CRDs are not installed, skipping adding to scheme."}
{"severity":"INFO","timestamp":"2024-05-07T16:23:35+02:00","logger":"setup","msg":"the env var WATCH_NAMESPACE isn't set, watching all namespaces"}
{"severity":"INFO","timestamp":"2024-05-07T16:23:35+02:00","msg":"Webhooks are disabled, operator is running an unsupported mode","ENABLE_WEBHOOKS":"false"}
{"severity":"INFO","timestamp":"2024-05-07T16:23:35+02:00","logger":"setup","msg":"starting manager"}
```
