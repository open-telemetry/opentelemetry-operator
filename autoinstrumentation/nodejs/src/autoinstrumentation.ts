import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-grpc';
import { PrometheusExporter } from '@opentelemetry/exporter-prometheus';
import { PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics';
import { alibabaCloudEcsDetector } from '@opentelemetry/resource-detector-alibaba-cloud';
import { awsEc2Detector, awsEksDetector } from '@opentelemetry/resource-detector-aws';
import { containerDetector } from '@opentelemetry/resource-detector-container';
import { gcpDetector } from '@opentelemetry/resource-detector-gcp';
import { envDetector, hostDetector, osDetector, processDetector } from '@opentelemetry/resources';
import { diag } from '@opentelemetry/api';

import { NodeSDK } from '@opentelemetry/sdk-node';

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
    instrumentations: [getNodeAutoInstrumentations()],
    traceExporter: new OTLPTraceExporter(),
    metricReader: getMetricReader(),
    resourceDetectors:
        [
            // Standard resource detectors.
            containerDetector,
            envDetector,
            hostDetector,
            osDetector,
            processDetector,

            // Cloud resource detectors.
            alibabaCloudEcsDetector,
            // Ordered AWS Resource Detectors as per:
            // https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/processor/resourcedetectionprocessor/README.md#ordering
            awsEksDetector,
            awsEc2Detector,
            gcpDetector,
        ],
});

sdk.start();
