#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

NEPTUNE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

${NEPTUNE_ROOT}/hack/generate-groups.sh "deepcopy,client,informer,lister" \
github.com/edgeai-neptune/neptune/pkg/client github.com/edgeai-neptune/neptune/pkg/apis \
"neptune:v1alpha1" \
--go-header-file ${NEPTUNE_ROOT}/hack/boilerplate/boilerplate.txt
