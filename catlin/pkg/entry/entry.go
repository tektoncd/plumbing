package entry

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// Entry represents a resource in the catalog, such as "task/git-clone" or
// "pipeline/build-and-deploy".
type Entry struct {
	fs fs.FS
}

// FromPath returns a catalog Entry from a given string path if that
// path leads to a correctly-structured catalog entry on disk, or an
// error if not.
func FromPath(resourcePath string) (*Entry, error) {
	entryFS := os.DirFS(resourcePath)
	if _, err := getLatestVersion(entryFS); err != nil {
		return nil, err
	}
	entry := &Entry{
		fs: entryFS,
	}
	return entry, nil
}

// GetLatestVersion returns the Version with the greatest major/minor
// values from the catalog entry's directory.
func (entry *Entry) GetLatestVersion() (Version, error) {
	if entry == nil {
		return Version{}, fmt.Errorf("invalid catalog entry")
	}
	return getLatestVersion(entry.fs)
}

func getLatestVersion(entryFS fs.FS) (Version, error) {
	versions, err := fs.ReadDir(entryFS, ".")
	if err != nil {
		return Version{}, fmt.Errorf("error reading catalog entry directory: %v", err)
	}
	latestVersion := zeroVersion
	for _, versionDir := range versions {
		name := versionDir.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !versionDir.IsDir() {
			continue
		}
		ver, err := ParseVersion(name)
		if err != nil {
			return latestVersion, fmt.Errorf("version parse error: %v", err)
		}
		if ver.Gt(latestVersion) {
			latestVersion = ver
		}
	}
	if latestVersion.Eq(zeroVersion) {
		return latestVersion, fmt.Errorf("no versions found")
	}
	return latestVersion, nil
}
