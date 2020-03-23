# Copyright 2018 The Knative Authors
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

FROM gcr.io/k8s-testimages/kubekins-e2e:v20190729-351ea95-master
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

# Install Go 1.14.1
ARG GO_VERSION=1.14.1
RUN rm -rf $(go env GOROOT) && \
    curl https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz > go${GO_VERSION}.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.tar.gz && \
    rm go${GO_VERSION}.tar.gz


# Install extras on top of base image
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update
RUN gcloud components update

# Docker
RUN gcloud components install docker-credential-gcr
RUN docker-credential-gcr configure-docker

# Extra tools through apt-get
RUN apt-get install -y uuid-runtime  # for uuidgen
RUN apt-get install -y rubygems  # for mdl

# Extra tools through go get
RUN go get -u github.com/google/ko/cmd/ko
RUN go get -u github.com/golang/dep/cmd/dep
RUN go get -u github.com/google/licenseclassifier
RUN go get -u github.com/google/go-licenses
RUN go get -u github.com/jstemmer/go-junit-report
RUN GO111MODULE="on" go get -u github.com/raviqqe/liche

# Install GolangCI linter: https://github.com/golangci/golangci-lint/
ARG GOLANGCI_VERSION=1.23.3
RUN curl -sL https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_VERSION}/golangci-lint-${GOLANGCI_VERSION}-linux-amd64.tar.gz | tar -C /usr/local/bin -xvzf - --strip-components=1 --wildcards "*/golangci-lint"

# Install the TektonCD CLI: https://github.com/tektoncd/cli/
ARG TKN_VERSION=0.7.1
RUN curl -sL https://github.com/tektoncd/cli/releases/download/v${TKN_VERSION}/tkn_${TKN_VERSION}_Linux_x86_64.tar.gz | tar -C /usr/local/bin -xvzf - --wildcards "tkn"

# Extra tools through gem
RUN gem install mixlib-config -v 2.2.4  # required because ruby is 2.1
RUN gem install mdl -v 0.5.0

# Extra tools through pip
RUN pip install yamllint

# Temporarily add test-infra to the image to build custom tools
RUN mkdir -p /go/src/github.com/knative/ && git clone https://github.com/knative/test-infra.git /go/src/github.com/knative/test-infra && \
    make -C /go/src/github.com/knative/test-infra/tools/githubhelper && \
    cp /go/src/github.com/knative/test-infra/tools/githubhelper/githubhelper /usr/local/bin && \
    go install github.com/knative/test-infra/tools/dep-collector
