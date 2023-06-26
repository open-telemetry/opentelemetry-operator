#!/bin/bash

sudo curl -Lo /usr/local/bin/kubectl-kuttl https://github.com/kudobuilder/kuttl/releases/download/v0.15.0/kubectl-kuttl_0.15.0_linux_x86_64
sudo chmod +x /usr/local/bin/kubectl-kuttl
export PATH=$PATH:/usr/local/bin
