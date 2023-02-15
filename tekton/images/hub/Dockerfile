# Copyright 2019 The Tekton Authors
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
FROM alpine:3.16@sha256:1bd67c81e4ad4b8f4a5c1c914d7985336f130e5cefb3e323654fd09d6bcdbbe2
LABEL maintainer "Tekton Authors <tekton-dev@googlegroups.com>"

RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories
RUN apk add --no-cache \
    bash \
    git=2.36.5-r0 \
    hub=2.14.2-r13
