# Metrics Basic Auth E2E Test App
Simple web application used in an end-to-end (E2E) test to verify that the OpenTelemetry collector can retrieve secret authentication details from the target allocator over mTLS.

## Overview
The web app provides a metrics endpoint secured with basic authentication, simulating real-world scenarios where services require secure access to their metrics. 

## Usage
This app is used within the E2E test suite to verify the OpenTelemetry operator's handling of mTLS-secured communications.