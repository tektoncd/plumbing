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

ARG GO_VERSION=1.23.4
FROM ghcr.io/ko-build/ko:v0.17.1@sha256:97b809c27679c4d1a6ed4a9acc18562a6b3688033cd3404c6e3722296cbdc802 AS ko
FROM golang:1.24.0-alpine@sha256:2d40d4fc278dad38be0777d5e2a88a2c6dee51b0b29c97a764fc6c6a11ca893c AS build
LABEL description="Build container"

RUN apk update && apk add --no-cache alpine-sdk ca-certificates
RUN update-ca-certificates

ARG KUBECTL_VERSION=1.28.11
RUN wget -O/usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v$KUBECTL_VERSION/bin/linux/amd64/kubectl; chmod +x /usr/local/bin/kubectl

# Install Kustomize
ENV GOBIN=/usr/local/go/bin
ENV GO111MODULE on
RUN go install sigs.k8s.io/kustomize/kustomize/v5@v5.4.2

# Install yq for YAML parsing
RUN apk add --no-cache yq

FROM google/cloud-sdk:512.0.0-alpine@sha256:888e4483c072117d288a5d6e2ebadee0653b259b2967ef21f77d9f7763a15a45
ARG GO_VERSION
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
LABEL org.opencontainers.image.source=https://github.com/tektoncd/plumbing
LABEL org.opencontainers.image.description="Build Container for Ko"
LABEL org.opencontainers.image.licenses=Apache-2.0

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
