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
FROM alpine/git:2.45.2@sha256:80ed34dbc31f821e1555812bfef429382b5d6959fc875c0937b44763ee780ebe
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

RUN adduser -D nonroot -u 1000
USER nonroot
