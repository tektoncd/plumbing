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

package validate

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tektoncd/plumbing/catlin/pkg/app"
	"github.com/tektoncd/plumbing/catlin/pkg/parser"
	"github.com/tektoncd/plumbing/catlin/pkg/validator"
)

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func validResourcePath() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires at least 1 path to tekton resource yaml but received none")
		}

		if !fileExists(args[0]) {
			return fmt.Errorf("no such file - %s", args[0])
		}
		return nil
	}
}

func Command(cli app.CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "validate",
		Aliases: []string{"verify"},
		Args:    validResourcePath(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validate(cli, args[0])
		},
	}
}

func validate(cli app.CLI, path string) error {

	r, err := os.Open(path)
	if err != nil {
		return err
	}

	// parse
	parser := parser.ForReader(r)
	res, err := parser.Parse()
	if err != nil {
		return err
	}

	// run validators
	validators := []validator.Validator{
		validator.NewPathValidator(cli, res, path),
		validator.NewContentValidator(cli, res),
		validator.ForKind(res),
	}

	result := validator.Result{}
	for _, v := range validators {
		result.Append(v.Validate())
	}

	// print result
	out := cli.Stream().Out
	for _, v := range result.Lints {
		switch v.Kind {
		case validator.Error:
			fmt.Fprintf(out, "ERROR: %s\n", v.Message)
		case validator.Warning:
			fmt.Fprintf(out, "WARN : %s\n", v.Message)
		case validator.Recommendation:
			fmt.Fprintf(out, "HINT : %s\n", v.Message)
		case validator.Info:
			fmt.Fprintf(out, "INFO : %s\n", v.Message)
		default:
			level := strings.ToUpper(string(v.Kind))
			fmt.Fprintf(out, "%s : %s\n", level, v.Message)
		}
	}
	if result.Errors != 0 {
		return fmt.Errorf("%s failed validation", path)
	}

	return nil
}
