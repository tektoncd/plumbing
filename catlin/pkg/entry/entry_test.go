package entry

import (
	"testing"
	"testing/fstest"
)

func TestGetLatestVersion(t *testing.T) {
	for _, tc := range []struct {
		name            string
		entry           Entry
		expectedVersion Version
	}{{
		name: "single catalog entry version parsed correctly",
		entry: Entry{
			fs: fstest.MapFS{
				"0.1/test.yaml": &fstest.MapFile{},
			},
		},
		expectedVersion: Version{0, 1},
	}, {
		name: "multiple catalog entry versions returns latest correctly",
		entry: Entry{
			fs: fstest.MapFS{
				"0.2/test.yaml": &fstest.MapFile{},
				"0.3/test.yaml": &fstest.MapFile{},
				"0.1/test.yaml": &fstest.MapFile{},
			},
		},
		expectedVersion: Version{0, 3},
	}, {
		name: "non-consecutive catalog entry versions returns latest correctly",
		entry: Entry{
			fs: fstest.MapFS{
				"4.8/test.yaml": &fstest.MapFile{},
				"0.2/test.yaml": &fstest.MapFile{},
				"0.1/test.yaml": &fstest.MapFile{},
			},
		},
		expectedVersion: Version{4, 8},
	}, {
		name: "catalog entry with versions starting after 0.1 returns latest correctly",
		entry: Entry{
			fs: fstest.MapFS{
				"2.91/test.yaml":    &fstest.MapFile{},
				"13.7/test.yaml":    &fstest.MapFile{},
				"1.11111/test.yaml": &fstest.MapFile{},
			},
		},
		expectedVersion: Version{13, 7},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			entryVersion, err := tc.entry.GetLatestVersion()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !entryVersion.Eq(tc.expectedVersion) {
				t.Errorf("expected version %s received %s", tc.expectedVersion, entryVersion)
			}
		})
	}
}
