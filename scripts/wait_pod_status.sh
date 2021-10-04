#!/bin/bash

set -euo pipefail

readonly NAME=$1
readonly STATUS=$2
readonly MAX_ATTEMPTS=30
readonly SLEEP_SEC=3
readonly KUBE_CTX=kind-kind

i=0

while :
do
  if [[ $i -gt $MAX_ATTEMPTS ]]; then
    echo "Max attempts exceeded: $i times"
    kubectl --context=${KUBE_CTX} get pods
    exit 1
  fi

  sts=($(\
    kubectl --context=${KUBE_CTX} get pods -o json | \
    jq -r ".items[] | select(.metadata.name | contains(\"${NAME}\")) | .status.phase" | \
    sort | uniq))

  if [[ ${#sts[@]} -eq 1 && $sts = $STATUS ]]; then
    break
  fi

  echo "Waiting for ${NAME} to be ${STATUS}..."
  sleep ${SLEEP_SEC}

  : $((++i))
done

exit 0
