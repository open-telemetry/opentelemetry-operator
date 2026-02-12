#!/usr/bin/env bash

# 对 Community bundle CSV 进行后处理修改
# 用法: ./hack/alauda-patch.sh <operator2-image-tag> <collector-image-tag>
# 示例: ./hack/alauda-patch.sh v0.144.0-r0 0.145.0-r0

set -euo pipefail

OPERATOR2_TAG="${1:?用法: $0 <operator2-image-tag> <collector-image-tag>}"
COLLECTOR_TAG="${2:?用法: $0 <operator2-image-tag> <collector-image-tag>}"

PATCH_FILE="alauda/alauda-csv.yaml"
CSV_FILE="bundle/community/manifests/opentelemetry-operator.clusterserviceversion.yaml"

# 检查 yq 是否可用
if ! command -v yq &> /dev/null; then
  echo "错误: 需要 yq 工具。"
  exit 1
fi

echo "Patch bundle CSV: ${CSV_FILE}"
echo "  OpenTelemetry Operator 镜像 tag: ${OPERATOR2_TAG}"
echo "  OpenTelemetry Collector 镜像 tag: ${COLLECTOR_TAG}"

# 1. Patch CSV
# 1.1 合并除 spec.install.spec.deployments 以外的所有部分
yq -i '
  . *= (load("'"${PATCH_FILE}"'") | del(.spec.install.spec.deployments))
' "${CSV_FILE}"

# 1.2 追加 spec.install.spec.deployments 中的环境变量
yq -i '
  with(.spec.install.spec.deployments[] | select(.name == "opentelemetry-operator-controller-manager").spec.template.spec.containers[] | select(.name == "manager"); 
    .env += (load("'"${PATCH_FILE}"'").spec.install.spec.deployments[] | select(.name == "opentelemetry-operator-controller-manager").spec.template.spec.containers[] | select(.name == "manager").env)
  )
' "${CSV_FILE}"

# 2. 替换 CSV 中的 TAG 占位符
# 使用 sed -i.bak 以兼容 macOS 和 Linux，随后删除备份文件
sed -i.bak "s|OPENTELEMETRY-OPERATOR2-TAG-PLACEHOLDER|${OPERATOR2_TAG}|g" "${CSV_FILE}"
sed -i.bak "s|OPENTELEMETRY-COLLECTOR-TAG-PLACEHOLDER|${COLLECTOR_TAG}|g" "${CSV_FILE}"
rm "${CSV_FILE}.bak"
