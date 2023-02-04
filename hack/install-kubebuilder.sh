#!/bin/bash

os=$(go env GOOS)
arch=$(go env GOARCH)
version=3.9.0
sudo curl -L https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${version}/kubebuilder_${os}_${arch} -o /usr/local/bin/kubebuilder
export PATH=$PATH:/usr/local/bin