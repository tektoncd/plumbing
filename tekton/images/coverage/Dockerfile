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

FROM golang:1.26.3@sha256:2d6c80227255c3112a4d08e67ba98e58efd3846daf15d9d7d4c389565d881b1a as buildcoverage
RUN git clone https://github.com/knative/test-infra /go/src/knative.dev/test-infra
RUN git -C /go/src/knative.dev/test-infra checkout ba4c2c3e061a59ac4b167da84924e9fa55475ad9 # Last commit before removal of tools/coverage
RUN make -C /go/src/knative.dev/test-infra/tools/coverage

FROM golang:1.26.3@sha256:2d6c80227255c3112a4d08e67ba98e58efd3846daf15d9d7d4c389565d881b1a
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
LABEL org.opencontainers.image.source=https://github.com/tektoncd/plumbing
LABEL org.opencontainers.image.description="Image for testing Coverage"
LABEL org.opencontainers.image.licenses=Apache-2.0

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git

COPY --from=buildcoverage /go/src/knative.dev/test-infra/tools/coverage/coverage /

ENTRYPOINT ["/coverage"]
