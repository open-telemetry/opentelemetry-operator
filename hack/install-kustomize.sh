#!/bin/bash

sudo curl -sL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.2.0/kustomize_v4.2.0_linux_amd64.tar.gz | tar xvz -C /usr/local/bin/
export PATH=$PATH:/usr/local/bin
