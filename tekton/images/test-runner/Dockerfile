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

ARG GO_VERSION=1.23.4
# Build kubetest independently of the rest
FROM golang:${GO_VERSION}@sha256:9820aca42262f58451f006de3213055974b36f24b31508c1baa73c967fcecb99 as kubetest
RUN git clone https://github.com/kubernetes/test-infra /go/src/k8s.io/test-infra
# Using e685556b32c5fb7ab12c3277d41112d47ceac0cd because after that, the URL kubetest
# uses needs extract credentials.
# FIXME: use kind and kops to run provision e2e tests clusters instead of kubetest
RUN cd /go/src/k8s.io/test-infra && \
    git checkout e685556b32c5fb7ab12c3277d41112d47ceac0cd && \
    go install k8s.io/test-infra/kubetest

FROM docker.io/library/ubuntu:24.04@sha256:c35e29c9450151419d9448b0fd75374fec4fff364a27f176fb458d472dfc9e54
ARG GO_VERSION
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
LABEL org.opencontainers.image.source=https://github.com/tektoncd/plumbing
LABEL org.opencontainers.image.description="Image for Test runner"
LABEL org.opencontainers.image.licenses=Apache-2.0

ENV TERM=xterm

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
    python3 \
    python3-dev \
    python3-openssl \
    python3-pip \
    python3-setuptools \
    python3-wheel \
    yamllint \
    rsync \
    unzip \
    wget \
    xz-utils \
    zip \
    zlib1g-dev \
    && rm -rf /var/lib/apt/lists/* \
    && update-alternatives --install /usr/bin/python python /usr/bin/python3 1 \
    && update-alternatives --install /usr/bin/pip pip /usr/bin/pip3 1

# Install gcloud

ENV PATH=/google-cloud-sdk/bin:/workspace:${PATH} \
    CLOUDSDK_CORE_DISABLE_PROMPTS=1

# To deal with "error: The gcp auth plugin has been removed" in Google Cloud SDK 393.0.0, use the gke-gcloud-auth-plugin.
# See https://cloud.google.com/blog/products/containers-kubernetes/kubectl-auth-changes-in-gke
ENV USE_GKE_GCLOUD_AUTH_PLUGIN=true

RUN wget -q https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz && \
    tar xzf google-cloud-sdk.tar.gz -C / && \
    rm google-cloud-sdk.tar.gz && \
    /google-cloud-sdk/install.sh \
    --disable-installation-options \
    --bash-completion=false \
    --path-update=false \
    --usage-reporting=false && \
    gcloud components update && \
    gcloud components install alpha beta kubectl docker-credential-gcr gke-gcloud-auth-plugin && \
    gcloud info | tee /gcloud-info.txt
RUN docker-credential-gcr configure-docker

#
# BEGIN: DOCKER IN DOCKER SETUP
# Install Docker deps, some of there are already installed in the image but
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

# Add the Docker apt-repository and Install Docker
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt update && apt install -y --no-install-recommends docker-ce && \
    rm -rf /var/lib/apt/lists/*

# TODO: the `sed` is a bit of a hack, look into alternatives.
# Why this exists: `docker service start` on debian runs a `cgroupfs_mount` method,
# We're already inside docker though so we can be sure these are already mounted.
# Trying to remount these makes for a very noisy error block in the beginning of
# the pod logs, so we just comment out the call to it... :shrug:
RUN sed -i 's/cgroupfs_mount$/#cgroupfs_mount\n/' /etc/init.d/docker \
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

# Go standard envs
ENV GOPATH /home/prow/go
ENV GOBIN /usr/local/go/bin
ENV PATH /usr/local/go/bin:$PATH

# preinstall:
# - bc for shell to junit
RUN apt update && apt install -y bc && \
    rm -rf /var/lib/apt/lists/*

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

# END: test-infra import

# Install Go
RUN curl https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz > go${GO_VERSION}.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.tar.gz && \
    rm go${GO_VERSION}.tar.gz
ENV GOROOT /usr/local/go

# Extra tools through apt
RUN apt update && apt install -y uuid-runtime  # for uuidgen
RUN apt update && apt install -y rubygems  # for mdl

# Install ko
ARG KO_VERSION=0.15.4
RUN curl -L https://github.com/google/ko/releases/download/v${KO_VERSION}/ko_${KO_VERSION}_Linux_x86_64.tar.gz > ko_${KO_VERSION}.tar.gz
RUN tar -C /usr/local/bin -xzf ko_${KO_VERSION}.tar.gz

# Extra tools through go install
ARG KIND_VERSION="v0.23.0"
RUN GO111MODULE="on" go install github.com/google/go-licenses@v1.0.0 && \
    GO111MODULE="on" go install github.com/jstemmer/go-junit-report@latest && \
    GO111MODULE="on" go install github.com/raviqqe/liche@latest && \
    GO111MODULE="on" go install sigs.k8s.io/kind@${KIND_VERSION}

# Install GolangCI linter: https://github.com/golangci/golangci-lint/
ARG GOLANGCI_VERSION=2.1.2
RUN curl -sL https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_VERSION}/golangci-lint-${GOLANGCI_VERSION}-linux-amd64.tar.gz | tar -C /usr/local/bin -xvzf - --strip-components=1 --wildcards "*/golangci-lint"

# Install Kustomize:
ARG KUSTOMIZE_VERSION=5.3.0
RUN curl -sL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz | tar -C /usr/local/bin -xvzf - --strip-components=1 --wildcards "kustomize"

# Install the TektonCD CLI: https://github.com/tektoncd/cli/
ARG TKN_VERSION=0.42.0
RUN curl -sL https://github.com/tektoncd/cli/releases/download/v${TKN_VERSION}/tkn_${TKN_VERSION}_Linux_x86_64.tar.gz | tar -C /usr/local/bin -xvzf - --wildcards "tkn"

# Extra tools through gem
RUN gem install mixlib-config -v 2.2.4  # required because ruby is 2.1
RUN gem install mdl -v 0.5.0

# Install yq
ARG YQ_VERSION=4.42.1
RUN wget https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_amd64 -O /usr/local/bin/yq && \
    chmod +x /usr/local/bin/yq

COPY --from=kubetest /go/bin/kubetest /usr/local/bin

# note the runner is also responsible for making docker in docker function if
# env DOCKER_IN_DOCKER_ENABLED is set and similarly responsible for generating
COPY ["entrypoint.sh", "runner.sh", "/usr/local/bin/"]
COPY setup-kind.sh /usr/local/bin/kind-e2e

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
