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
FROM ghcr.io/ko-build/ko:v0.17.1@sha256:97b809c27679c4d1a6ed4a9acc18562a6b3688033cd3404c6e3722296cbdc802 AS ko
FROM golang:1.24.0-alpine@sha256:2d40d4fc278dad38be0777d5e2a88a2c6dee51b0b29c97a764fc6c6a11ca893c
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
LABEL org.opencontainers.image.source=https://github.com/tektoncd/plumbing
LABEL org.opencontainers.image.description="Image for Ko"
LABEL org.opencontainers.image.licenses=Apache-2.0

RUN apk add --no-cache musl-dev gcc git
ENV GO111MODULE on
RUN go install sigs.k8s.io/kustomize/kustomize/v5@v5.4.2
COPY --from=ko /ko-app/ko /usr/local/bin/ko
