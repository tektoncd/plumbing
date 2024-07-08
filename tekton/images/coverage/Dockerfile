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

FROM golang:1.22@sha256:fcae9e0e7313c6467a7c6632ebb5e5fab99bd39bd5eb6ee34a211353e647827a as buildcoverage
RUN git clone https://github.com/knative/test-infra /go/src/knative.dev/test-infra
RUN git -C /go/src/knative.dev/test-infra checkout ba4c2c3e061a59ac4b167da84924e9fa55475ad9 # Last commit before removal of tools/coverage
RUN make -C /go/src/knative.dev/test-infra/tools/coverage

FROM golang:1.22@sha256:fcae9e0e7313c6467a7c6632ebb5e5fab99bd39bd5eb6ee34a211353e647827a
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git

COPY --from=buildcoverage /go/src/knative.dev/test-infra/tools/coverage/coverage /

ENTRYPOINT ["/coverage"]
