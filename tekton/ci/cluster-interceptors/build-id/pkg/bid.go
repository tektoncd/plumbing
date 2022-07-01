/*
 Copyright 2022 The Tekton Authors

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package pkg

import (
	"context"

	"github.com/bwmarrin/snowflake"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"

	"knative.dev/pkg/logging"
)

const (
	bidExtensionKey = "build-id"
	bidContentKey   = "id"
)

var (
	_    triggersv1.InterceptorInterface = (*Interceptor)(nil)
	node *snowflake.Node
)

type Interceptor struct{}
type buildIdNodeKey struct{}

func Get(ctx context.Context) *snowflake.Node {
	untyped := ctx.Value(buildIdNodeKey{})
	if untyped == nil {
		logging.FromContext(ctx).Errorf("Unable to fetch node from context.")
		return nil
	}
	return untyped.(*snowflake.Node)
}

// ToContext adds the cloud events client to the context
func ToContext(ctx context.Context, n *snowflake.Node) context.Context {
	return context.WithValue(ctx, buildIdNodeKey{}, n)
}

func (w Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	// Get a build ID
	node = Get(ctx)
	return &triggersv1.InterceptorResponse{
		Extensions: map[string]interface{}{
			bidExtensionKey: map[string]interface{}{
				bidContentKey: node.Generate().String(),
			},
		},
		Continue: true,
	}
}
