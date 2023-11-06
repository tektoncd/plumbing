# Copyright 2019 The Tekton Authors
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
FROM golang:1.21-alpine@sha256:96a8a701943e7eabd81ebd0963540ad660e29c3b2dc7fb9d7e06af34409e9ba6 as build
LABEL description="Build container"

ARG TKN_VERSION=0.32.2
RUN ARCH=$(uname -m) \
    OS=$(uname) \
    && wget -O- https://github.com/tektoncd/cli/releases/download/v${TKN_VERSION}/tkn_${TKN_VERSION}_${OS}_${ARCH}.tar.gz | tar zxf - -C /usr/local/bin

# Copy tkn binary from build container
FROM alpine:3.18@sha256:eece025e432126ce23f223450a0326fbebde39cdf496a85d8c016293fc851978
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
COPY --from=build /usr/local/bin/tkn /usr/local/bin/tkn

