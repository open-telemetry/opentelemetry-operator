#!/bin/bash
# Since operator-sdk 1.26.0, `make bundle` changes the `createdAt` field from the bundle
# even if it is patched:
#   https://github.com/operator-framework/operator-sdk/pull/6136
# This code checks if only the createdAt field. If is the only change, it is ignored.
# Else, it will do nothing.
# https://github.com/operator-framework/operator-sdk/issues/6285#issuecomment-1415350333
git diff --quiet -I'^    createdAt: ' bundle/manifests/opentelemetry-operator.clusterserviceversion.yaml
ret=$?
changes=$(git diff --numstat bundle/manifests/opentelemetry-operator.clusterserviceversion.yaml)
if [ $ret = 0 ] && [ "$changes" = '1	1	bundle/manifests/opentelemetry-operator.clusterserviceversion.yaml' ] ; then
    git checkout bundle/manifests/opentelemetry-operator.clusterserviceversion.yaml
fi
