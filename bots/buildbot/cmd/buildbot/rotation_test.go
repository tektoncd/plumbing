package main

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

func fromFile(file string) GetFile {
	return func() (io.ReadCloser, error) {
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("Could not open file %s: %v", file, err)
		}
		return f, nil
	}
}

func TestGetBuildCaptain(t *testing.T) {
	r := NewRotation(fromFile("testdata/rotation.csv"))

	for _, c := range []struct {
		desc            string
		time            time.Time
		expectedCaptain string
	}{{
		desc:            "success",
		time:            time.Date(2019, time.December, 4, 0, 0, 0, 0, time.UTC),
		expectedCaptain: "christiewilson",
	}, {
		desc:            "no one on the date",
		time:            time.Date(2019, time.December, 15, 0, 0, 0, 0, time.UTC),
		expectedCaptain: "",
	}, {
		desc:            "time not found",
		time:            time.Date(2016, time.August, 15, 0, 0, 0, 0, time.UTC),
		expectedCaptain: "nobody",
	}} {
		t.Run(c.desc, func(t *testing.T) {
			captain := r.GetBuildCaptain(c.time)
			if captain != c.expectedCaptain {
				t.Errorf("Expected build captain %s for %s but got %s", c.expectedCaptain, c.time, captain)
			}
		})
	}
}

func TestGetBuildCaptain_InvalidFile(t *testing.T) {
	r := NewRotation(fromFile("testdata/rotation-invalid.csv"))
	captain := r.GetBuildCaptain(time.Date(2019, time.December, 4, 0, 0, 0, 0, time.UTC))
	if captain != "nobody" {
		t.Errorf("Expected build captain nobody when file is invalid but got %s", captain)
	}
}

func TestGetBuildCaptain_MissingFile(t *testing.T) {
	r := NewRotation(fromFile("testdata/rotation-missing.csv"))
	captain := r.GetBuildCaptain(time.Date(2019, time.December, 4, 0, 0, 0, 0, time.UTC))
	if captain != "nobody" {
		t.Errorf("Expected build captain nobody when file is not found but got %s", captain)
	}
}
