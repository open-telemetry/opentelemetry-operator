// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import _ "embed"

// Init container scripts for Apache HTTPD and Nginx auto-instrumentation.
// They live in scripts/*.sh so they can be linted with shellcheck and read
// without unescaping Go string literals. User-controlled values (config paths
// and file names from the Instrumentation CRD) are passed as positional
// arguments.
//
//go:embed scripts/apache_httpd_agent.sh
var apacheHttpdAgentScript string

//go:embed scripts/nginx_clone.sh
var nginxCloneScript string

//go:embed scripts/nginx_agent.sh
var nginxAgentScript string
