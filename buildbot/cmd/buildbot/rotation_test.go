package main

import (
	"testing"
	"time"
)

func TestGetBuildCop(t *testing.T) {
	r := NewRotation("testdata/rotation.csv")

	for _, c := range []struct {
		desc        string
		time        time.Time
		expectedCop string
	}{{
		desc:        "success",
		time:        time.Date(2019, time.December, 4, 0, 0, 0, 0, time.UTC),
		expectedCop: "christiewilson",
	}, {
		desc:        "no one on the date",
		time:        time.Date(2019, time.December, 15, 0, 0, 0, 0, time.UTC),
		expectedCop: "",
	}, {
		desc:        "time not found",
		time:        time.Date(2016, time.August, 15, 0, 0, 0, 0, time.UTC),
		expectedCop: "nobody",
	}} {
		t.Run(c.desc, func(t *testing.T) {
			cop := r.GetBuildCop(c.time)
			if cop != c.expectedCop {
				t.Errorf("Expected build cop %s for %s but got %s", c.expectedCop, c.time, cop)
			}
		})
	}
}

func TestGetBuildCop_Invalid(t *testing.T) {
	r := NewRotation("testdata/rotation-invalid.csv")
	cop := r.GetBuildCop(time.Date(2019, time.December, 4, 0, 0, 0, 0, time.UTC))
	if cop != "nobody" {
		t.Errorf("Expected build cop nobody when file is invalid but got %s", cop)
	}
}
