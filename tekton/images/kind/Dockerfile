# Copyright 2021 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Runtime image for common dependencies when running kind tests.

FROM golang:1.19-alpine@sha256:31c62d91ada60d52cd2e40e1134778d32549cd58c131b30dd41069f998dc8490

WORKDIR /kind

RUN GO111MODULE=on go install github.com/google/ko@v0.12.0

RUN apk add --no-cache \
  bash curl docker git jq openssl

# Install kubectl and make sure it's available in the PATH.
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && chmod +x ./kubectl && mv ./kubectl /bin

RUN curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.10.0/kind-$(uname)-amd64" && chmod +x ./kind && mv ./kind /bin
