#!/bin/sh
# Clone-container init script for PHP auto-instrumentation.
# Runs as a clone of the application container so it can detect PHP API version,
# PHP extension directory, glibc / musl standard C library and thread safety. Copy the result
# into a shared volume for the agent init container to pick up.
#
# Inputs:
#   $1 - Directory containing detected PHP API version, PHP extension directory, glibc / musl standard C library and thread safety (e.g. /otel-auto-instrumentation-php-clone)
set -e

clone="$1"

thread_safety=non-zts
if [ "enabled" = "$(php -i | grep "^Thread Safety => " | awk '{print $4}')" ]; then
    thread_safety=zts
fi
echo "$thread_safety" > "$clone"/thread_safety.txt

api=$(php -i | grep "^PHP API => " | awk '{print $4}')
echo "$api" > "$clone"/api.txt

# check if alpine
standard_c_lib=glibc
if [ -f /etc/alpine-release ]; then
    standard_c_lib=musl
fi
echo "$standard_c_lib" > "$clone"/standard_c_lib.txt
