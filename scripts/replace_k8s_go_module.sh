#!/bin/bash

set -euo pipefail

# @see https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-521493597

readonly VERSION=$1
readonly MODS=($(
  curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/go.mod |
  sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p'
))

readonly SIZE=${#MODS[@]}
i=1

for mod in "${MODS[@]}"; do
  ver=$(
    go mod download -json "${mod}@kubernetes-${VERSION}" |
    sed -n 's|.*"Version": "\(.*\)".*|\1|p'
  )

  echo "$(date '+%Y-%m-%d %H:%M:%S.%3N') ${mod}@${ver} ... $((i++))/${SIZE}"
  go mod edit "-replace=${mod}=${mod}@${ver}"
done
