# Copyright 2018 The Tekton Authors
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
FROM ghcr.io/ko-build/ko:v0.16.0@sha256:2e1108263f0c5f6bff2961b3101162319a77930c4377a654ea179c9cd14f7d1e AS ko
FROM golang:1.23-alpine@sha256:d0b31558e6b3e4cc59f6011d79905835108c919143ebecc58f35965bf79948f4
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

RUN apk add --no-cache musl-dev gcc git
ENV GO111MODULE on
RUN go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.7
COPY --from=ko /ko-app/ko /usr/local/bin/ko
