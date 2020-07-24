// Copyright © 2020 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"io"
	"os"

	"go.uber.org/zap"
)

type Stream struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

type CLI interface {
	Logger() *zap.Logger
	Stream() *Stream
}

type cli struct {
	log    *zap.Logger
	stream *Stream
}

var _ CLI = (*cli)(nil)

func New() *cli {
	log, _ := zap.NewProduction()
	return &cli{
		log:    log,
		stream: &Stream{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}}
}

func (c *cli) Logger() *zap.Logger {
	return c.log
}

func (c *cli) Stream() *Stream {
	return c.stream
}
