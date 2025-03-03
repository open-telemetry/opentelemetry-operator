#!/bin/sh

# SPDX-FileCopyrightText: Copyright 2025 The OpenTelemetry authors
# SPDX-License-Identifier: Apache-2.0

set -eu

if [ "${OTEL_LOG_LEVEL:-}" = "debug" ]; then
  set -x
fi

cd -P -- "$(dirname -- "$0")"

if [ -z "${INSTRUMENTATION_FOLDER_SOURCE:-}" ]; then
  INSTRUMENTATION_FOLDER_SOURCE=/instrumentation
fi
if [ ! -d "${INSTRUMENTATION_FOLDER_SOURCE}" ]; then
  >&2 echo "[OTel injector] Instrumentation source directory ${INSTRUMENTATION_FOLDER_SOURCE} does not exist."
  exit 1
fi

if [ -z "${INSTRUMENTATION_FOLDER_DESTINATION:-}" ]; then
  INSTRUMENTATION_FOLDER_DESTINATION=/
fi

# We deliberately do not create the base directory for $INSTRUMENTATION_FOLDER_DESTINATION via mkdir, it needs
# be an existing mount point provided externally.
cp -R "${INSTRUMENTATION_FOLDER_SOURCE}"/ "${INSTRUMENTATION_FOLDER_DESTINATION}"

if [ "${OTEL_DEBUG_DEBUG:-}" = "debug" ]; then
  >&2 echo "[OTel injector] Status of '${INSTRUMENTATION_FOLDER_DESTINATION}' after copying instrumentation files: $(find ${INSTRUMENTATION_FOLDER_DESTINATION})"
fi
