// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package otlpvalidator_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const truncateLength = 100 // Define truncation length as a constant

// Helper function to read the file line by line and process each line.
func processOtlpFile(t *testing.T, filePath string) (foundTraces, foundMetrics, foundLogs bool) {
	t.Helper()

	file, err := os.Open(filePath)
	require.NoError(t, err, "Failed to open file: %s", filePath)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	// Create unmarshalers.
	traceUnmarshaler := &ptrace.JSONUnmarshaler{}
	metricUnmarshaler := &pmetric.JSONUnmarshaler{}
	logUnmarshaler := &plog.JSONUnmarshaler{}

	for scanner.Scan() {
		lineNumber++
		lineBytes := scanner.Bytes()
		lineStr := scanner.Text() // For logging context if needed.

		if len(strings.TrimSpace(lineStr)) == 0 {
			fmt.Printf("Skipping empty line %d\n", lineNumber)
			continue
		}

		processedLine := false

		// Try Traces.
		td, err := traceUnmarshaler.UnmarshalTraces(lineBytes)
		// Check for unmarshal error *AND* that the structure actually contains data.
		if err == nil && td.ResourceSpans().Len() > 0 {
			fmt.Printf("Line %d: Processing TRACES batch (SpanCount: %d)\n", lineNumber, td.SpanCount())
			validateTraces(t, td, lineNumber)
			foundTraces = true // Mark traces as found.
			processedLine = true
		} else if err != nil {
			// Log the trace unmarshal error specifically if you need more detail.
			t.Logf("Line %d: Failed to unmarshal as Traces: %v. Content preview: %s", lineNumber, err, truncate(lineStr))
		}

		// Try Metrics (only if not processed as traces).
		if !processedLine {
			md, err := metricUnmarshaler.UnmarshalMetrics(lineBytes)
			if err == nil && md.ResourceMetrics().Len() > 0 {
				fmt.Printf("Line %d: Processing METRICS batch (DataPointCount: %d)\n", lineNumber, md.DataPointCount())
				validateMetrics(t, md, lineNumber)
				foundMetrics = true // Mark metrics as found.
				processedLine = true
			} else if err != nil {
				t.Logf("Line %d: Failed to unmarshal as Metrics: %v. Content preview: %s", lineNumber, err, truncate(lineStr))
			}
		}

		// Try Logs (only if not processed as traces or metrics).
		if !processedLine {
			ld, err := logUnmarshaler.UnmarshalLogs(lineBytes)
			if err == nil && ld.ResourceLogs().Len() > 0 {
				fmt.Printf("Line %d: Processing LOGS batch (LogRecordCount: %d)\n", lineNumber, ld.LogRecordCount())
				validateLogs(t, ld, lineNumber)
				foundLogs = true // Mark logs as found.
				processedLine = true
			} else if err != nil {
				t.Logf("Line %d: Failed to unmarshal as Logs: %v. Content preview: %s", lineNumber, err, truncate(lineStr))
			}
		}

		if !processedLine {
			// This indicates either an unmarshal error for all types or an empty batch for all types.
			assert.Fail(t, "Failed to process line or line contained empty batch", "Line %d content preview: %s", lineNumber, truncate(lineStr))
		}
	}

	require.NoError(t, scanner.Err(), "Error reading file: %s", filePath)
	return // Return the found flags.
}

// --- Validation Functions ---

// validateResourceAttributes checks resource attributes.
func validateResourceAttributes(t *testing.T, res pcommon.Resource, lineNum int, signalType string) {
	t.Helper()
	attrs := res.Attributes()
	assert.Greater(t, attrs.Len(), 0, "Line %d (%s): Resource attributes should not be empty", lineNum, signalType)

	// Example: Assert specific resource attributes exist and have correct type/value.
	serviceName, ok := attrs.Get("service.name")
	assert.True(t, ok, "Line %d (%s): Resource missing 'service.name'", lineNum, signalType)
	if ok {
		assert.Equal(t, pcommon.ValueTypeStr, serviceName.Type(), "Line %d (%s): 'service.name' attribute type should be string, got %s", lineNum, signalType, serviceName.Type().String())
		if serviceName.Type() == pcommon.ValueTypeStr {
			assert.NotEmpty(t, serviceName.Str(), "Line %d (%s): 'service.name' attribute should not be empty", lineNum, signalType)
		}
	}

	hostArch, ok := attrs.Get("host.arch")
	assert.True(t, ok, "Line %d (%s): Resource missing 'host.arch'", lineNum, signalType)
	if ok {
		assert.Equal(t, pcommon.ValueTypeStr, hostArch.Type(), "Line %d (%s): 'host.arch' attribute type should be string, got %s", lineNum, signalType, hostArch.Type().String())
		if hostArch.Type() == pcommon.ValueTypeStr {
			assert.NotEmpty(t, hostArch.Str(), "Line %d (%s): 'host.arch' attribute should not be empty", lineNum, signalType)
		}
	}
	processPid, ok := attrs.Get("process.pid")
	assert.True(t, ok, "Line %d (%s): Resource missing 'process.pid'", lineNum, signalType)
	if ok {
		assert.Equal(t, pcommon.ValueTypeInt, processPid.Type(), "Line %d (%s): 'process.pid' attribute type should be int, got %s", lineNum, signalType, processPid.Type().String())
		if processPid.Type() == pcommon.ValueTypeInt {
			assert.Greater(t, processPid.Int(), int64(0), "Line %d (%s): 'process.pid' should be positive", lineNum, signalType)
		}
	}

	attrs.Range(func(k string, v pcommon.Value) bool {
		assert.NotEqual(t, pcommon.ValueTypeEmpty, v.Type(), "Line %d (%s): Resource attribute '%s' has empty/invalid type", lineNum, signalType, k)
		return true
	})
}

// validateSignalAttributes checks attributes within a span, metric datapoint, or log record.
func validateSignalAttributes(t *testing.T, attrs pcommon.Map, lineNum int, signalType, context string) {
	t.Helper()
	attrs.Range(func(k string, v pcommon.Value) bool {
		assert.NotEqual(t, pcommon.ValueTypeEmpty, v.Type(), "Line %d (%s): Attribute '%s' in '%s' has empty/invalid type", lineNum, signalType, k, context)
		return true
	})

	// Example: Check 'http.status_code' on a trace span, expecting an Int.
	if signalType == "trace" {
		statusCode, ok := attrs.Get("http.status_code")
		if ok {
			assert.Equal(t, pcommon.ValueTypeInt, statusCode.Type(), "Line %d (%s): Attribute 'http.status_code' should be int, got %s (%s)", lineNum, signalType, statusCode.Type().String(), context)
			if statusCode.Type() == pcommon.ValueTypeInt {
				assert.Greater(t, statusCode.Int(), int64(0), "Line %d (%s): 'http.status_code' should be positive (%s)", lineNum, signalType, context)
			}
		}
	}

	// Example: Check 'daemon' attribute on a metric data point, expecting Bool.
	if signalType == "metric" {
		daemonAttr, ok := attrs.Get("daemon")
		if ok {
			assert.Equal(t, pcommon.ValueTypeBool, daemonAttr.Type(), "Line %d (%s): Attribute 'daemon' should be bool, got %s (%s)", lineNum, signalType, daemonAttr.Type().String(), context)
		}
	}
}

// validateTraces performs basic trace validation and attribute checks.
func validateTraces(t *testing.T, td ptrace.Traces, lineNum int) {
	t.Helper()
	rs := td.ResourceSpans()
	require.Greater(t, rs.Len(), 0, "Line %d: Trace data unmarshaled but ResourceSpans slice is empty", lineNum)

	for i := 0; i < rs.Len(); i++ {
		res := rs.At(i).Resource()
		validateResourceAttributes(t, res, lineNum, "trace")

		scopeSpans := rs.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				spanCtx := fmt.Sprintf("Resource %d, Scope %d, Span %d (%s)", i, j, k, span.Name())

				assert.False(t, span.TraceID().IsEmpty(), "Line %d: Span TraceID is empty (%s)", lineNum, spanCtx)
				assert.False(t, span.SpanID().IsEmpty(), "Line %d: Span SpanID is empty (%s)", lineNum, spanCtx)
				assert.NotZero(t, span.StartTimestamp(), "Line %d: Span StartTimestamp is zero (%s)", lineNum, spanCtx)
				assert.NotZero(t, span.EndTimestamp(), "Line %d: Span EndTimestamp is zero (%s)", lineNum, spanCtx)
				assert.GreaterOrEqual(t, span.EndTimestamp(), span.StartTimestamp(), "Line %d: Span EndTimestamp < StartTimestamp (%s)", lineNum, spanCtx)
				assert.NotEmpty(t, span.Name(), "Line %d: Span Name is empty (%s)", lineNum, spanCtx)

				validateSignalAttributes(t, span.Attributes(), lineNum, "trace", spanCtx)
			}
		}
	}
}

// validateMetrics performs basic metric validation and attribute checks.
func validateMetrics(t *testing.T, md pmetric.Metrics, lineNum int) {
	t.Helper()
	rm := md.ResourceMetrics()
	require.Greater(t, rm.Len(), 0, "Line %d: Metric data unmarshaled but ResourceMetrics slice is empty", lineNum)

	for i := 0; i < rm.Len(); i++ {
		res := rm.At(i).Resource()
		validateResourceAttributes(t, res, lineNum, "metric")

		scopeMetrics := rm.At(i).ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			metrics := scopeMetrics.At(j).Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				metricCtx := fmt.Sprintf("Resource %d, Scope %d, Metric %d (%s)", i, j, k, metric.Name())

				assert.NotEmpty(t, metric.Name(), "Line %d: Metric name is empty (%s)", lineNum, metricCtx)
				//assert.NotEqual(t, pmetric.MetricTypeEmpty, metric.Type(), "Line %d: Metric type is Empty (%s)", lineNum, metricCtx)

				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					dps := metric.Gauge().DataPoints()
					for dpIdx := 0; dpIdx < dps.Len(); dpIdx++ {
						dp := dps.At(dpIdx)
						dpCtx := fmt.Sprintf("%s, DataPoint %d", metricCtx, dpIdx)
						assert.NotZero(t, dp.Timestamp(), "Line %d: Gauge DP Timestamp is zero (%s)", lineNum, dpCtx)
						validateSignalAttributes(t, dp.Attributes(), lineNum, "metric", dpCtx)
					}
				case pmetric.MetricTypeSum:
					sum := metric.Sum()
					assert.NotEqual(t, pmetric.AggregationTemporalityUnspecified, sum.AggregationTemporality(), "Line %d: Sum AggregationTemporality is Unspecified (%s)", lineNum, metricCtx)
					dps := sum.DataPoints()
					for dpIdx := 0; dpIdx < dps.Len(); dpIdx++ {
						dp := dps.At(dpIdx)
						dpCtx := fmt.Sprintf("%s, DataPoint %d", metricCtx, dpIdx)
						assert.NotZero(t, dp.StartTimestamp(), "Line %d: Sum DP StartTimestamp is zero (%s)", lineNum, dpCtx)
						assert.NotZero(t, dp.Timestamp(), "Line %d: Sum DP Timestamp is zero (%s)", lineNum, dpCtx)
						assert.GreaterOrEqual(t, dp.Timestamp(), dp.StartTimestamp(), "Line %d: Sum DP Timestamp < StartTimestamp (%s)", lineNum, dpCtx)
						validateSignalAttributes(t, dp.Attributes(), lineNum, "metric", dpCtx)
					}
				case pmetric.MetricTypeHistogram:
					hist := metric.Histogram()
					assert.NotEqual(t, pmetric.AggregationTemporalityUnspecified, hist.AggregationTemporality(), "Line %d: Histogram AggregationTemporality is Unspecified (%s)", lineNum, metricCtx)
					dps := hist.DataPoints()
					for dpIdx := 0; dpIdx < dps.Len(); dpIdx++ {
						dp := dps.At(dpIdx)
						dpCtx := fmt.Sprintf("%s, DataPoint %d", metricCtx, dpIdx)
						assert.NotZero(t, dp.StartTimestamp(), "Line %d: Histogram DP StartTimestamp is zero (%s)", lineNum, dpCtx)
						assert.NotZero(t, dp.Timestamp(), "Line %d: Histogram DP Timestamp is zero (%s)", lineNum, dpCtx)
						assert.GreaterOrEqual(t, dp.Timestamp(), dp.StartTimestamp(), "Line %d: Histogram DP Timestamp < StartTimestamp (%s)", lineNum, dpCtx)
						validateSignalAttributes(t, dp.Attributes(), lineNum, "metric", dpCtx)
					}
				case pmetric.MetricTypeEmpty:
					t.Logf("Line %d: Metric type is Empty (%s)", lineNum, metricCtx)
				case pmetric.MetricTypeExponentialHistogram:
					t.Logf("Line %d: Metric type is ExponentialHistogram - validation not fully implemented (%s)", lineNum, metricCtx)
					// Add specific validation for ExponentialHistogram if needed.
					hist := metric.ExponentialHistogram()
					assert.NotEqual(t, pmetric.AggregationTemporalityUnspecified, hist.AggregationTemporality(), "Line %d: ExponentialHistogram AggregationTemporality is Unspecified (%s)", lineNum, metricCtx)
					dps := hist.DataPoints()
					for dpIdx := 0; dpIdx < dps.Len(); dpIdx++ {
						dp := dps.At(dpIdx)
						dpCtx := fmt.Sprintf("%s, DataPoint %d", metricCtx, dpIdx)
						assert.NotZero(t, dp.StartTimestamp(), "Line %d: ExponentialHistogram DP StartTimestamp is zero (%s)", lineNum, dpCtx)
						assert.NotZero(t, dp.Timestamp(), "Line %d: ExponentialHistogram DP Timestamp is zero (%s)", lineNum, dpCtx)
						assert.GreaterOrEqual(t, dp.Timestamp(), dp.StartTimestamp(), "Line %d: ExponentialHistogram DP Timestamp < StartTimestamp (%s)", lineNum, dpCtx)
						validateSignalAttributes(t, dp.Attributes(), lineNum, "metric", dpCtx)
					}
				case pmetric.MetricTypeSummary:
					t.Logf("Line %d: Metric type is Summary - validation not fully implemented (%s)", lineNum, metricCtx)
					// Add specific validation for Summary if needed.
					dps := metric.Summary().DataPoints()
					for dpIdx := 0; dpIdx < dps.Len(); dpIdx++ {
						dp := dps.At(dpIdx)
						dpCtx := fmt.Sprintf("%s, DataPoint %d", metricCtx, dpIdx)
						assert.NotZero(t, dp.StartTimestamp(), "Line %d: Summary DP StartTimestamp is zero (%s)", lineNum, dpCtx)
						assert.NotZero(t, dp.Timestamp(), "Line %d: Summary DP Timestamp is zero (%s)", lineNum, dpCtx)
						assert.GreaterOrEqual(t, dp.Timestamp(), dp.StartTimestamp(), "Line %d: Summary DP Timestamp < StartTimestamp (%s)", lineNum, dpCtx)
						validateSignalAttributes(t, dp.Attributes(), lineNum, "metric", dpCtx)
					}
				default:
					// This case should theoretically not be reached if all types are handled.
					t.Errorf("Line %d: Unknown metric type %s encountered (%s)", lineNum, metric.Type().String(), metricCtx)
				}
			}
		}
	}
}

// validateLogs performs basic log validation and attribute checks.
func validateLogs(t *testing.T, ld plog.Logs, lineNum int) {
	t.Helper()
	rl := ld.ResourceLogs()
	require.Greater(t, rl.Len(), 0, "Line %d: Log data unmarshaled but ResourceLogs slice is empty", lineNum)

	for i := 0; i < rl.Len(); i++ {
		res := rl.At(i).Resource()
		validateResourceAttributes(t, res, lineNum, "log")

		scopeLogs := rl.At(i).ScopeLogs()
		for j := 0; j < scopeLogs.Len(); j++ {
			logRecords := scopeLogs.At(j).LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				lr := logRecords.At(k)
				logCtx := fmt.Sprintf("Resource %d, Scope %d, LogRecord %d", i, j, k)

				assert.NotZero(t, lr.ObservedTimestamp(), "Line %d: LogRecord ObservedTimestamp is zero (%s)", lineNum, logCtx)
				if lr.Timestamp() != 0 {
					assert.Greater(t, lr.Timestamp(), pcommon.Timestamp(0), "Line %d: LogRecord Timestamp is zero (%s)", lineNum, logCtx)
				}
				assert.NotEqual(t, pcommon.ValueTypeEmpty, lr.Body().Type(), "Line %d: LogRecord Body is empty (%s)", lineNum, logCtx)

				validateSignalAttributes(t, lr.Attributes(), lineNum, "log", logCtx)
			}
		}
	}
}

// Helper to truncate strings for logging.
// Removed maxLen parameter to satisfy 'unparam' linter.
func truncate(s string) string {
	if len(s) <= truncateLength {
		return s
	}
	return s[:truncateLength] + "..."
}

// --- Test Execution ---

// TestValidateOtlpFile is the main test function.
func TestValidateOtlpFile(t *testing.T) {
	// Get file path from environment variable.
	filePath := os.Getenv("OTLP_FILE_PATH")
	if filePath == "" {
		t.Skip("Skipping test: OTLP_FILE_PATH environment variable not set")
		return
	}
	t.Logf("Using OTLP file path from environment variable: %s", filePath)

	// Get expected telemetry types from environment variable.
	telemetryCheck := os.Getenv("TELEMETRY_CHECK")
	expectedSignals := make(map[string]bool)
	if telemetryCheck != "" {
		types := strings.Split(telemetryCheck, ",")
		for _, tt := range types {
			trimmedType := strings.TrimSpace(tt)
			if trimmedType != "" {
				expectedSignals[trimmedType] = true // Mark this type as expected.
			}
		}
		t.Logf("Expecting telemetry types based on TELEMETRY_CHECK: %v", expectedSignals)
	} else {
		t.Logf("TELEMETRY_CHECK not set, will not enforce presence of specific signal types.")
	}

	// Check if the file exists before processing.
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		t.Fatalf("File specified by OTLP_FILE_PATH does not exist: %s", filePath)
	}
	require.NoError(t, err, "Error checking file status: %s", filePath)

	// Process the file and get flags indicating which signals were found.
	foundTraces, foundMetrics, foundLogs := processOtlpFile(t, filePath)

	// Assert that all *expected* telemetry types were found.
	if len(expectedSignals) > 0 { // Only check if TELEMETRY_CHECK was set.
		if expectedSignals["traces"] {
			assert.True(t, foundTraces, "Expected traces based on TELEMETRY_CHECK, but none were found in the file.")
		}
		if expectedSignals["metrics"] {
			assert.True(t, foundMetrics, "Expected metrics based on TELEMETRY_CHECK, but none were found in the file.")
		}
		if expectedSignals["logs"] {
			assert.True(t, foundLogs, "Expected logs based on TELEMETRY_CHECK, but none were found in the file.")
		}
	}
}
