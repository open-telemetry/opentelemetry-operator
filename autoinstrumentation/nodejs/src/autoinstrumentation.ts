import { getNodeAutoInstrumentations, getResourceDetectorsFromEnv } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter as OTLPProtoTraceExporter } from '@opentelemetry/exporter-trace-otlp-proto';
import { OTLPTraceExporter as OTLPHttpTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { OTLPTraceExporter as OTLPGrpcTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-grpc';
import { PrometheusExporter } from '@opentelemetry/exporter-prometheus';
import { PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics';
import { diag, DiagConsoleLogger } from '@opentelemetry/api';

import { NodeSDK, core } from '@opentelemetry/sdk-node';

diag.setLogger(
    new DiagConsoleLogger(),
    core.getEnv().OTEL_LOG_LEVEL
);


function getTraceExporter() {
    let protocol = process.env.OTEL_EXPORTER_OTLP_PROTOCOL;
    switch (protocol) {
        case undefined:
        case '':
        case 'grpc':
            return new OTLPGrpcTraceExporter();
        case 'http/json':
            return new OTLPHttpTraceExporter();
        case 'http/protobuf':
            return new OTLPProtoTraceExporter();
        default:
            throw Error(`Creating traces exporter based on "${protocol}" protocol (configured via environment variable OTEL_EXPORTER_OTLP_PROTOCOL) is not implemented!`);
    }
}

function getMetricReader() {
    switch (process.env.OTEL_METRICS_EXPORTER) {
        case undefined:
        case '':
        case 'otlp':
            diag.info('using otel metrics exporter');
            return new PeriodicExportingMetricReader({
                exporter: new OTLPMetricExporter(),
            });
        case 'prometheus':
            diag.info('using prometheus metrics exporter');
            return new PrometheusExporter({});
        case 'none':
            diag.info('disabling metrics reader');
            return undefined;
        default:
            throw Error(`no valid option for OTEL_METRICS_EXPORTER: ${process.env.OTEL_METRICS_EXPORTER}`)
    }
}

const sdk = new NodeSDK({
    autoDetectResources: true,
    instrumentations: getNodeAutoInstrumentations(),
    traceExporter: getTraceExporter(),
    metricReader: getMetricReader(),
    resourceDetectors: getResourceDetectorsFromEnv()
});

try {
    sdk.start();
    diag.info('OpenTelemetry automatic instrumentation started successfully');
} catch (error) {
    diag.error(
        'Error initializing OpenTelemetry SDK. Your application is not instrumented and will not produce telemetry',
        error
    );
}

async function shutdown(): Promise<void> {
    try {
        await sdk.shutdown();
        diag.debug('OpenTelemetry SDK terminated');
    } catch (error) {
        diag.error('Error terminating OpenTelemetry SDK', error);
    }
}

// Gracefully shutdown SDK if a SIGTERM is received
process.on('SIGTERM', shutdown);
// Gracefully shutdown SDK if Node.js is exiting normally
process.once('beforeExit', shutdown);
