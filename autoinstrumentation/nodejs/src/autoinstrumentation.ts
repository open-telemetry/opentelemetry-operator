import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { alibabaCloudEcsDetector } from '@opentelemetry/resource-detector-alibaba-cloud';
import {
    awsBeanstalkDetector,
    awsEc2Detector,
    awsEcsDetector,
    awsEksDetector,
    awsLambdaDetector
} from '@opentelemetry/resource-detector-aws';
import { containerDetector } from '@opentelemetry/resource-detector-container';
import { gcpDetector } from '@opentelemetry/resource-detector-gcp';
import { gitHubDetector } from '@opentelemetry/resource-detector-github';
import { browserDetector, envDetector, hostDetector, osDetector, processDetector } from '@opentelemetry/resources';
import { instanaAgentDetector } from '@opentelemetry/resource-detector-instana';

import { NodeSDK } from '@opentelemetry/sdk-node';

const sdk = new NodeSDK({
    autoDetectResources: true,
    instrumentations: [getNodeAutoInstrumentations()],
    traceExporter: new OTLPTraceExporter(),
    resourceDetectors:
        [
            // Standard resource detectors.
            browserDetector,
            containerDetector,
            envDetector,
            hostDetector,
            osDetector,
            processDetector,

            // Cloud resource detectors.
            alibabaCloudEcsDetector,
            awsBeanstalkDetector,
            awsEc2Detector,
            awsEcsDetector,
            awsEksDetector,
            awsLambdaDetector,
            gcpDetector,
            gitHubDetector,

            // Agent resource detectors.
            instanaAgentDetector,
        ],
});

sdk.start();
