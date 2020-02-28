package main

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateRotationCSV(t *testing.T) {
	for _, tc := range []struct {
		description string
		config      config
		expected    string
	}{{
		description: "blank names are rendered on weekends",
		config: config{
			names:     strings.Split("bob,carol,eric", ","),
			days:      4,
			startDate: parseDate(t, "2020-02-28"),
			startName: "eric",
		},
		expected: `
Date,User
2020-02-28,eric
2020-02-29,
2020-03-01,
2020-03-02,bob
`,
	}, {
		description: "names are repeated until rotation's end",
		config: config{
			names:     strings.Split("bob,carol", ","),
			days:      4,
			startDate: parseDate(t, "2020-03-02"),
			startName: "bob",
		}, expected: `
Date,User
2020-03-02,bob
2020-03-03,carol
2020-03-04,bob
2020-03-05,carol
`,
	}} {
		t.Run(tc.description, func(t *testing.T) {
			var out strings.Builder
			if err := generateRotationCSV(tc.config, &out); err != nil {
				t.Errorf("unable to generate csv: %v", err)
			}
			s := strings.TrimSpace(out.String())
			exp := strings.TrimSpace(tc.expected)
			if s != exp {
				t.Errorf("--- EXPECTED ---\n%v\n--- RECEIVED ---\n%v", exp, s)
			}
		})
	}
}

func parseDate(t *testing.T, s string) time.Time {
	parsedTime, err := time.Parse(expectedDateFormat, s)
	if err != nil {
		t.Fatalf("error parsing test case date %q, format must be YYYY-MM-DD: %v", s, err)
	}
	return parsedTime
}
