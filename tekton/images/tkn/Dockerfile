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
FROM golang:1.23.6-alpine@sha256:2c49857f2295e89b23b28386e57e018a86620a8fede5003900f2d138ba9c4037 as build
LABEL description="Build container"

ARG TKN_VERSION=0.39.1
RUN ARCH=$(uname -m) \
    OS=$(uname) \
    && wget -O- https://github.com/tektoncd/cli/releases/download/v${TKN_VERSION}/tkn_${TKN_VERSION}_${OS}_${ARCH}.tar.gz | tar zxf - -C /usr/local/bin

# Copy tkn binary from build container
FROM alpine:3.21@sha256:56fa17d2a7e7f168a043a2712e63aed1f8543aeafdcee47c58dcffe38ed51099
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
LABEL org.opencontainers.image.source=https://github.com/tektoncd/plumbing
LABEL org.opencontainers.image.description="Image for tkn"
LABEL org.opencontainers.image.licenses=Apache-2.0
COPY --from=build /usr/local/bin/tkn /usr/local/bin/tkn

