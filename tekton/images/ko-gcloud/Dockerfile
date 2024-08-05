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

ARG GO_VERSION=1.22.1
FROM ghcr.io/ko-build/ko:v0.16.0@sha256:2e1108263f0c5f6bff2961b3101162319a77930c4377a654ea179c9cd14f7d1e AS ko
FROM golang:1.22-alpine@sha256:0d3653dd6f35159ec6e3d10263a42372f6f194c3dea0b35235d72aabde86486e AS build
LABEL description="Build container"

RUN apk update && apk add --no-cache alpine-sdk ca-certificates
RUN update-ca-certificates

ARG KUBECTL_VERSION=1.22.13
RUN wget -O/usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v$KUBECTL_VERSION/bin/linux/amd64/kubectl; chmod +x /usr/local/bin/kubectl

# Install Kustomize
ENV GOBIN=/usr/local/go/bin
ENV GO111MODULE on
RUN go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.7

# Install yq for YAML parsing
RUN apk add --no-cache yq

FROM google/cloud-sdk:486.0.0-alpine@sha256:4c437f46e93d26af92e62c8d549c04d6deb15e559363fc8ccc2d1ccbbbbd0431
ARG GO_VERSION
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

# Install golang
RUN curl https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz > go${GO_VERSION}.tar.gz
RUN tar -C /usr/local -xzf go${GO_VERSION}.tar.gz
ENV PATH="${PATH}:/usr/local/go/bin"
ENV GOROOT /usr/local/go

# Get static binaries from the build image
COPY --from=ko /ko-app/ko /usr/local/bin/ko
COPY --from=build /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=build /usr/local/go/bin/kustomize /usr/local/go/bin
COPY --from=build /usr/bin/yq /usr/local/bin/yq
