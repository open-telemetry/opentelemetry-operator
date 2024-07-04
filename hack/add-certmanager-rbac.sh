#!/bin/bash

kubectl apply -f - <<EOF
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    app.kubernetes.io/name: opentelemetry-operator
  name: opentelemetry-operator-manager-certmanager-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: opentelemetry-operator-manager-certmanager-role
subjects:
  - kind: ServiceAccount
    name: opentelemetry-operator-controller-manager
    namespace: opentelemetry-operator-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: opentelemetry-operator-manager-certmanager-role
rules:
- apiGroups:
  - cert-manager.io
  resources:
  - issuers
  - certificaterequests
  - certificates
  verbs:
  - create
  - get
  - list
  - watch
  - update
  - patch
  - delete
EOF
