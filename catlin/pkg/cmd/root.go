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

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tektoncd/plumbing/catlin/pkg/app"
	"github.com/tektoncd/plumbing/catlin/pkg/cmd/bump"
	"github.com/tektoncd/plumbing/catlin/pkg/cmd/linter"
	"github.com/tektoncd/plumbing/catlin/pkg/cmd/validate"
)

func Root(cli app.CLI) *cobra.Command {
	var cmd = &cobra.Command{
		Use:          "catlin",
		Short:        "Lints Tekton Resources and Catalogs",
		Long:         ``,
		SilenceUsage: true,
	}

	cmd.AddCommand(
		validate.Command(cli),
		linter.Command(cli),
		bump.Command(cli),
	)

	return cmd
}
