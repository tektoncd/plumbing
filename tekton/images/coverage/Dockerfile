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

FROM golang:1.16.3 as buildcoverage
RUN git clone https://github.com/knative/test-infra /go/src/knative.dev/test-infra
RUN make -C /go/src/knative.dev/test-infra/tools/coverage

FROM golang:1.16.3
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git

COPY --from=buildcoverage /go/src/knative.dev/test-infra/tools/coverage/coverage /

ENTRYPOINT ["/coverage"]
