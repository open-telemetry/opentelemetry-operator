#!/usr/bin/env bash
# Since operator-sdk 1.26.0, `make bundle` changes the `createdAt` field from the bundle
# even if it is patched:
#   https://github.com/operator-framework/operator-sdk/pull/6136
# This code checks if only the createdAt field. If is the only change, it is ignored.
# Else, it will do nothing.
# https://github.com/operator-framework/operator-sdk/issues/6285#issuecomment-1415350333
# --no-ext-diff ensures a globally configured external diff driver (e.g. difftastic)
# does not bypass the -I regex, which would leave the createdAt-only change unstripped.
git diff --no-ext-diff --quiet -I'^    createdAt: ' bundle
if ((! $?)) ; then
    git checkout bundle
fi
