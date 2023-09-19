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
FROM golang:1.21-alpine@sha256:96634e55b363cb93d39f78fb18aa64abc7f96d372c176660d7b8b6118939d97b as build
LABEL description="Build container"

ARG TKN_VERSION=0.32.0
RUN ARCH=$(uname -m) \
    OS=$(uname) \
    && wget -O- https://github.com/tektoncd/cli/releases/download/v${TKN_VERSION}/tkn_${TKN_VERSION}_${OS}_${ARCH}.tar.gz | tar zxf - -C /usr/local/bin

# Copy tkn binary from build container
FROM alpine:3.16@sha256:1bd67c81e4ad4b8f4a5c1c914d7985336f130e5cefb3e323654fd09d6bcdbbe2
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
COPY --from=build /usr/local/bin/tkn /usr/local/bin/tkn

