#!/bin/bash

os=$(go env GOOS)
arch=$(go env GOARCH)
version=4.5.7
sudo curl -sL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${version}/kustomize_v${version}_${os}_${arch}.tar.gz | tar xvz -C /usr/local/bin/
export PATH=$PATH:/usr/local/bin
