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
FROM golang:1.14-alpine3.12 as build
LABEL description="Build container"

RUN apk update && apk add --no-cache alpine-sdk ca-certificates
RUN update-ca-certificates

ARG KUBECTL_VERSION=1.18.1
RUN wget -O/usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v$KUBECTL_VERSION/bin/linux/amd64/kubectl; chmod +x /usr/local/bin/kubectl

# Install Kustomize
ENV GOBIN=/usr/local/go/bin
ENV GO111MODULE on
RUN go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4

FROM google/cloud-sdk:296.0.0-alpine
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

# Install golang
RUN curl https://dl.google.com/go/go1.14.linux-amd64.tar.gz > go1.14.tar.gz
RUN tar -C /usr/local -xzf go1.14.tar.gz
ENV PATH="${PATH}:/usr/local/go/bin"

# Install ko
ENV GOBIN=/usr/local/go/bin
ENV GO111MODULE on
RUN go get github.com/google/ko/cmd/ko@v0.5.1

# Get Kubectl and Kustomize from the build image
COPY --from=build /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=build /usr/local/go/bin/kustomize /usr/local/go/bin
