import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-grpc';
import { PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics';
import { alibabaCloudEcsDetector } from '@opentelemetry/resource-detector-alibaba-cloud';
import { awsEc2Detector, awsEksDetector } from '@opentelemetry/resource-detector-aws';
import { containerDetector } from '@opentelemetry/resource-detector-container';
import { gcpDetector } from '@opentelemetry/resource-detector-gcp';
import { envDetector, hostDetector, osDetector, processDetector } from '@opentelemetry/resources';

import { NodeSDK } from '@opentelemetry/sdk-node';

const sdk = new NodeSDK({
    autoDetectResources: true,
    instrumentations: [getNodeAutoInstrumentations()],
    traceExporter: new OTLPTraceExporter(),
    metricReader: new PeriodicExportingMetricReader({
        exporter: new OTLPMetricExporter(),
    }),
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
