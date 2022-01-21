package entry

import (
	"fmt"
	"strconv"
	"strings"
)

var zeroVersion = Version{0, 0}

// Version represents a catalog entry's version in "major.minor" format.
type Version struct {
	Major int64
	Minor int64
}

func (rv Version) Eq(other Version) bool {
	return rv.Major == other.Major && rv.Minor == other.Minor
}

func (rv Version) Lt(other Version) bool {
	if rv.Major < other.Major {
		return true
	}
	return rv.Major == other.Major && rv.Minor < other.Minor
}

func (rv Version) Gt(other Version) bool {
	if rv.Major > other.Major {
		return true
	}
	return rv.Major == other.Major && rv.Minor > other.Minor
}

func (rv Version) String() string {
	return fmt.Sprintf("%d.%d", rv.Major, rv.Minor)
}

func (rv Version) BumpMinor() Version {
	return Version{
		Major: rv.Major,
		Minor: rv.Minor + 1,
	}
}

func ParseVersion(from string) (Version, error) {
	parsed := Version{}
	version := strings.SplitN(from, ".", 2)
	if len(version) != 2 {
		return parsed, fmt.Errorf("incorrectly formatted version directory %q", from)
	}
	var err error
	if parsed.Major, err = strconv.ParseInt(version[0], 10, 64); err != nil {
		return parsed, fmt.Errorf("error parsing major version from %q: %v", from, err)
	}
	if parsed.Minor, err = strconv.ParseInt(version[1], 10, 64); err != nil {
		return parsed, fmt.Errorf("error parsing minor version from %q: %v", from, err)
	}
	return parsed, nil
}
