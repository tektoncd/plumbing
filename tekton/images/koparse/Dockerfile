# Copyright 2021 The Tekton Authors
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
FROM python:3.15.0a3-alpine@sha256:c7fcf93b725e5e2e9c506f1c20630e3617907cf9e68236a6873d2954a05bd9c1
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
LABEL org.opencontainers.image.source=https://github.com/tektoncd/plumbing
LABEL org.opencontainers.image.description="Image for koparse"
LABEL org.opencontainers.image.licenses=Apache-2.0

COPY ["koparse/koparse.py", "/usr/local/bin/koparse"]
RUN chmod +x /usr/local/bin/koparse
ENTRYPOINT ["/usr/local/bin/koparse"]
