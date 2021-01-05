#!/usr/bin/env bash

###
#Copyright 2019 The KubeEdge Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.
###

set -o errexit
set -o nounset
set -o pipefail

NEPTUNE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

${NEPTUNE_ROOT}/hack/update-vendor.sh
 
if git status --short 2>/dev/null | grep -qE 'go\.mod|go\.sum|vendor/'; then
  echo "FAILED: vendor verify failed." >&2
  echo "Please run the command to update your vendor directories: hack/update-vendor.sh" >&2
  exit 1
else
  echo "SUCCESS: vendor verified."
fi
