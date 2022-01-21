package bump

import (
	"fmt"
	"path/filepath"

	dircp "github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"github.com/tektoncd/plumbing/catlin/pkg/app"
	"github.com/tektoncd/plumbing/catlin/pkg/entry"
)

func Command(p app.CLI) *cobra.Command {
	return &cobra.Command{
		Use:     "bump",
		Args:    validResourcePath(),
		Short:   "Bump version of existing catalog entry",
		Long:    "Creates a new version of an existing catalog entry by copying its latest version into a new directory",
		Example: "catlin bump task/ansible-runner",
		RunE:    bumpVersion,
	}
}

func bumpVersion(cmd *cobra.Command, args []string) error {
	entryPath := args[0]
	ent, err := entry.FromPath(args[0])
	if err != nil {
		return fmt.Errorf("invalid catalog entry: %v", err)
	}
	latestVersion, err := ent.GetLatestVersion()
	if err != nil {
		return fmt.Errorf("error reading latest version of catalog entry: %v", err)
	}
	oldVersionPath := filepath.Join(entryPath, latestVersion.String())
	newVersionPath := filepath.Join(entryPath, latestVersion.BumpMinor().String())
	fmt.Printf("Copying %s to %s\n", oldVersionPath, newVersionPath)
	dircp.Copy(oldVersionPath, newVersionPath)
	return nil
}

func validResourcePath() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf(`requires exactly one path to an existing catalog entry. e.g. "./task/foo-bar"`)
		}

		if _, err := entry.FromPath(args[0]); err != nil {
			return fmt.Errorf("invalid catalog entry: %v", err)
		}
		return nil
	}
}
