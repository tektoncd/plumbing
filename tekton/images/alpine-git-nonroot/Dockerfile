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
FROM alpine/git:2.45.1@sha256:d73869d3c6615a1a0c9616b0532c81e8fe1f536af7a774c87b647d4edcc8e4a7
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

RUN adduser -D nonroot -u 1000
USER nonroot
