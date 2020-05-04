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
FROM google/cloud-sdk:latest
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

# Install golang
RUN curl https://dl.google.com/go/go1.13.linux-amd64.tar.gz > go1.13.tar.gz
RUN tar -C /usr/local -xzf go1.13.tar.gz
ENV PATH="${PATH}:/usr/local/go/bin"

# Install ko
ENV GOBIN=/usr/local/go/bin
ENV GO111MODULE on
RUN go get github.com/google/ko/cmd/ko@v0.5.0

# Install Kustomize
RUN go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4
