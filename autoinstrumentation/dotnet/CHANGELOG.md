# Changelog — .NET auto-instrumentation image

All notable changes to the OpenTelemetry .NET auto-instrumentation image
(`ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-dotnet`) are
recorded here, one entry per published `<sdk-version>-<revision>` image tag. See
[../README.md](../README.md) for the tagging scheme.

Entries are added automatically by the autoinstrumentation-revision tooling when
the image's SDK version or contents change; see [../README.md](../README.md).

## 1.17.0-1

- Update .NET auto-instrumentation from 1.16.0 to 1.17.0. See [release notes](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/tag/v1.17.0). (#5652)
- **Breaking change**: in the `OpenTelemetry.Instrumentation.Process` instrumentation, the `process.cpu.time` metric attribute `process.cpu.state` was renamed to `cpu.mode`, and the metric description changed to "Total CPU seconds broken down by different CPU modes." See [open-telemetry/opentelemetry-dotnet-contrib#4602](https://github.com/open-telemetry/opentelemetry-dotnet-contrib/pull/4602).

## 1.16.0-1

- Initial changelog entry for the OpenTelemetry .NET auto-instrumentation image.
