#!/usr/bin/env bash
# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

DIR="${GOPATH}/src/github.com/google/licenseclassifier"

if [ -d "$DIR" ]; then
    echo "$DIR is not Empty. Skip clone..."
else
    # Workaround for https://github.com/google/licenseclassifier/issues/20
    git clone https://github.com/google/licenseclassifier $DIR
fi

cleanup_dind() {
    if [[ "{DOCKER_IN_DOCKER_ENABLED:-false}" == "true" ]]; then
        echo "Cleaning up after docker"
        docker ps -aq | xargs -r docker rm -f || true
        service docker stop || true
    fi
    # cleanup binfmt_misc
    echo "Cleaning up binfmt_misc ..."
}

# optionally enable ipv6 docker
export DOCKER_IN_DOCKER_IPV6_ENABLED=${DOCKER_IN_DOCKER_IPV6_ENABLED:-false}
if [[ "${DOCKER_IN_DOCKER_IPV6_ENABLED}" == "true" ]]; then
    echo "Enabling IPV6 for Docker."
    # configure the daemon with ipv6
    mkdir -p /etc/docker/
    cat <<EOF >/etc/docker/daemon.json
{
  "ipv6": true,
  "fixed-cidr-v6": "fc00:db8:1::/64"
}
EOF
    # enable ipv6
    sysctl net.ipv6.conf.all.disable_ipv6=0
    sysctl net.ipv6.conf.all.forwarding=1
    # enable ipv6 iptables
    modprobe -v ip6table_nat
fi

# disable error exit so we can run post-command cleanup
set +o errexit

# add $GOPATH/bin to $PATH
export PATH="${GOPATH}/bin:${PATH}"
mkdir -p "${GOPATH}/bin"
# Authenticate gcloud, allow failures
if [[ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]]; then
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}" || true
fi

# actually start bootstrap and the job
echo "== Running ./runner.sh backward compatibility test runner ==="
set -o xtrace

while [[ $# -ne 0 ]]; do
    case $1 in
        # FIXME(vdemeester) Remove those
        --scenario=*) ;;
        --clean) ;;
        --job=*) ;;
        --repo=*) ;;
        --root=*) ;;
        --upload=*) ;;
        --service-account=*)
            gcloud auth activate-service-account --key-file=$(cut -d "=" -f2 <<< "$1")
            ;;
        --)
            shift
            # Remove extra '--'
            [[ $1 == "--" ]] && shift
            break
            ;;
        *)
            echo "error: unknown option $1"
            exit 1
            ;;
    esac
    shift
done

export PIPELINE_DIR=https://github.com/tektoncd/pipeline

git clone
"$@"

EXIT_VALUE=$?
set +o xtrace

# cleanup after job
if [[ "${DOCKER_IN_DOCKER_ENABLED}" == "true" ]]; then
    echo "Cleaning up after docker in docker."
    printf '=%.0s' {1..80}; echo
    cleanup_dind
    printf '=%.0s' {1..80}; echo
    echo "Done cleaning up after docker in docker."
fi

# preserve exit value from job / bootstrap
exit ${EXIT_VALUE}
