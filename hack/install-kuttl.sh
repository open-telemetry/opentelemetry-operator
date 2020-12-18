#!/bin/bash

sudo curl -Lo /usr/local/bin/kubectl-kuttl https://github.com/kudobuilder/kuttl/releases/download/v0.7.2/kubectl-kuttl_0.7.2_linux_x86_64
sudo chmod +x /usr/local/bin/kubectl-kuttl
export PATH=$PATH:/usr/local/bin
