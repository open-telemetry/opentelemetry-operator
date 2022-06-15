#!/bin/bash

sudo curl -Lo /usr/local/bin/kubectl-kuttl https://github.com/kudobuilder/kuttl/releases/download/v0.12.1/kubectl-kuttl_0.12.1_linux_x86_64
sudo chmod +x /usr/local/bin/kubectl-kuttl
export PATH=$PATH:/usr/local/bin
