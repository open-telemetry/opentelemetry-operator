import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { getNodeAutoResourceDetectors } from '@opentelemetry/auto-resource-detectors-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';

import { NodeSDK } from '@opentelemetry/sdk-node';

const sdk = new NodeSDK({
    autoDetectResources: true,
    instrumentations: [getNodeAutoInstrumentations()],
    traceExporter: new OTLPTraceExporter(),
    resourceDetectors: [getNodeAutoResourceDetectors()],
});

sdk.start();
