# Copyright 2020 The Tekton Authors
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
FROM gcr.io/google.com/cloudsdktool/cloud-sdk:alpine@sha256:18963f32aebecb6c25f7340f56815b49307bc526f7efaa7c6f3a2eb73f7426d9
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
LABEL org.opencontainers.image.source=https://github.com/tektoncd/plumbing
LABEL org.opencontainers.image.description="Image for Buildx"
LABEL org.opencontainers.image.licenses=Apache-2.0

RUN  mkdir -p ~/.docker/cli-plugins \
     && curl -fsSL https://github.com/docker/buildx/releases/download/v0.8.1/buildx-v0.8.1.linux-amd64 > ~/.docker/cli-plugins/docker-buildx \
     && chmod u+x ~/.docker/cli-plugins/docker-buildx
