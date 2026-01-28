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
FROM python:3.15.0a5-alpine@sha256:f77a9303949324561403f4be60dca95b3c7e42f7e5510e78aa5564da3104d6e2
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"
LABEL org.opencontainers.image.source=https://github.com/tektoncd/plumbing
LABEL org.opencontainers.image.description="Image for koparse"
LABEL org.opencontainers.image.licenses=Apache-2.0

COPY ["koparse/koparse.py", "/usr/local/bin/koparse"]
RUN chmod +x /usr/local/bin/koparse
ENTRYPOINT ["/usr/local/bin/koparse"]
