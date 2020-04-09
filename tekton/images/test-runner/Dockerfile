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

FROM debian:buster
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

ENV DEBIAN_FRONTEND noninteractive

WORKDIR /workspace
RUN mkdir -p /workspace
ENV WORKSPACE=/workspace \
    TERM=xterm

# common util tools
# https://github.com/GoogleCloudPlatform/gsutil/issues/446 for python-openssl
RUN apt update && apt install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    file \
    git \
    jq \
    mercurial \
    openssh-client \
    pkg-config \
    procps \
    python \
    python-dev \
    python-openssl \
    python-pip \
    python3 \
    python3-dev \
    python3-openssl \
    python3-pip \
    rsync \
    unzip \
    wget \
    xz-utils \
    zip \
    zlib1g-dev \
    && rm -rf /var/lib/apt/lists/* \
    && update-alternatives --install /usr/bin/python python /usr/bin/python3 1 \
    && update-alternatives --install /usr/bin/pip pip /usr/bin/pip3 1 \
    && python -m pip install --upgrade pip setuptools wheel

# Install gcloud

ENV PATH=/google-cloud-sdk/bin:/workspace:${PATH} \
    CLOUDSDK_CORE_DISABLE_PROMPTS=1

RUN wget -q https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz && \
    tar xzf google-cloud-sdk.tar.gz -C / && \
    rm google-cloud-sdk.tar.gz && \
    /google-cloud-sdk/install.sh \
        --disable-installation-options \
        --bash-completion=false \
        --path-update=false \
        --usage-reporting=false && \
    gcloud components install alpha beta kubectl && \
    gcloud info | tee /workspace/gcloud-info.txt


#
# BEGIN: DOCKER IN DOCKER SETUP
#

# Install Docker deps, some of these are already installed in the image but
# that's fine since they won't re-install and we can reuse the code below
# for another image someday.
RUN apt update && apt install -y --no-install-recommends \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg2 \
    software-properties-common \
    lsb-release && \
    rm -rf /var/lib/apt/lists/*

# Add the Docker apt-repository
RUN curl -fsSL https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg \
    | apt-key add - && \
    add-apt-repository \
    "deb [arch=amd64] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") \
    $(lsb_release -cs) stable"

# Install Docker
# TODO: the `sed` is a bit of a hack, look into alternatives.
# Why this exists: `docker service start` on debian runs a `cgroupfs_mount` method,
# We're already inside docker though so we can be sure these are already mounted.
# Trying to remount these makes for a very noisy error block in the beginning of
# the pod logs, so we just comment out the call to it... :shrug:
RUN apt update && apt install -y --no-install-recommends docker-ce=5:19.03.* && \
    rm -rf /var/lib/apt/lists/* && \
    sed -i 's/cgroupfs_mount$/#cgroupfs_mount\n/' /etc/init.d/docker \
    && update-alternatives --set iptables /usr/sbin/iptables-legacy \
    && update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy



# Move Docker's storage location
RUN echo 'DOCKER_OPTS="${DOCKER_OPTS} --data-root=/docker-graph"' | \
    tee --append /etc/default/docker
# NOTE this should be mounted and persisted as a volume ideally (!)
# We will make a fallback one now just in case
RUN mkdir /docker-graph

#
# END: DOCKER IN DOCKER SETUP
#

# hint to kubetest that it is in CI
ENV KUBETEST_IN_DOCKER="true"
# Go standard envs
ENV GOPATH /go
ENV PATH /usr/local/go/bin:$PATH

RUN mkdir -p /go/bin
ENV PATH $GOPATH/bin:$PATH

# setup k8s repo symlink
RUN mkdir -p /go/src/k8s.io/kubernetes \
    && ln -s /go/src/k8s.io/kubernetes /workspace/kubernetes

# preinstall:
# - graphviz package for graphing profiles
# - bc for shell to junit
# - rpm for building RPMs with Bazel
RUN apt update && apt install -y bc \
    graphviz \
    rpm && \
    rm -rf /var/lib/apt/lists/*
# preinstall this for kops tests xref kubetest prepareAws(...)
# TODO(krzyzacy,justisb,chrislovecnm): remove this
RUN python -m pip install awscli

# install cfssl to prevent https://github.com/kubernetes/kubernetes/issues/55589
# The invocation at the end is to prevent download failures downloads as in the bug.
# TODO(porridge): bump CFSSL_VERSION to one where cfssljson supports the -version flag and test it as well.
ARG CFSSL_VERSION="R1.2"
RUN wget -q -O cfssl "https://pkg.cfssl.org/${CFSSL_VERSION}/cfssl_linux-amd64" && \
    wget -q -O cfssljson "https://pkg.cfssl.org/${CFSSL_VERSION}/cfssljson_linux-amd64" && \
    chmod +x cfssl cfssljson && \
    mv cfssl cfssljson /usr/local/bin && \
    cfssl version

# replace kubectl with one from K8S_RELEASE
ARG K8S_RELEASE=latest
RUN rm -f $(which kubectl) && \
    export KUBECTL_VERSION=$(curl https://storage.googleapis.com/kubernetes-release/release/${K8S_RELEASE}.txt) && \
    wget https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl -O /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl

# everything below will be triggered on every new image tag ...
ADD ["https://raw.githubusercontent.com/kubernetes/kubernetes/master/cluster/get-kube.sh", \
    "/workspace/"]
RUN ["/bin/chmod", "+x", "/workspace/get-kube.sh"]

# note the runner is also responsible for making docker in docker function if
# env DOCKER_IN_DOCKER_ENABLED is set and similarly responsible for generating
COPY ["entrypoint.sh", "runner.sh", "/usr/local/bin/"]

# END: test-infra import

# Install Go 1.14.1
ARG GO_VERSION=1.14.1
RUN rm -rf $(go env GOROOT) && \
    curl https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz > go${GO_VERSION}.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.tar.gz && \
    rm go${GO_VERSION}.tar.gz


# Install extras on top of base image
RUN gcloud components update

# Docker
RUN gcloud components install docker-credential-gcr
RUN docker-credential-gcr configure-docker

# Extra tools through apt
RUN apt update && apt install -y uuid-runtime  # for uuidgen
RUN apt update && apt install -y rubygems  # for mdl

# Extra tools through go get
RUN go get -u github.com/google/ko/cmd/ko
RUN go get -u github.com/golang/dep/cmd/dep
RUN go get -u github.com/google/licenseclassifier
RUN go get -u github.com/google/go-licenses
RUN go get -u github.com/jstemmer/go-junit-report
RUN GO111MODULE="on" go get -u github.com/raviqqe/liche

# Install GolangCI linter: https://github.com/golangci/golangci-lint/
ARG GOLANGCI_VERSION=1.24.0
RUN curl -sL https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_VERSION}/golangci-lint-${GOLANGCI_VERSION}-linux-amd64.tar.gz | tar -C /usr/local/bin -xvzf - --strip-components=1 --wildcards "*/golangci-lint"

# Install the TektonCD CLI: https://github.com/tektoncd/cli/
ARG TKN_VERSION=0.8.0
RUN curl -sL https://github.com/tektoncd/cli/releases/download/v${TKN_VERSION}/tkn_${TKN_VERSION}_Linux_x86_64.tar.gz | tar -C /usr/local/bin -xvzf - --wildcards "tkn"

# Extra tools through gem
RUN gem install mixlib-config -v 2.2.4  # required because ruby is 2.1
RUN gem install mdl -v 0.5.0

# Extra tools through pip
RUN python -m pip install yamllint

# Temporarily add test-infra to the image to build custom tools
RUN mkdir -p /go/src/github.com/knative/ && git clone https://github.com/knative/test-infra.git /go/src/github.com/knative/test-infra && \
    make -C /go/src/github.com/knative/test-infra/tools/githubhelper && \
    cp /go/src/github.com/knative/test-infra/tools/githubhelper/githubhelper /usr/local/bin && \
    go install github.com/knative/test-infra/tools/dep-collector

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
