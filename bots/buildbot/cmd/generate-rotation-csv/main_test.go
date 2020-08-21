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
	}, {
		description: "overrides usurp generated values",
		config: config{
			names:     strings.Split("ronda,carol,eric,dan", ","),
			days:      4,
			startDate: parseDate(t, "2020-03-02"),
			startName: "ronda",
			overrides: map[string]string{
				"2020-03-03": "dan",
				"2020-03-04": "",
			},
		}, expected: `
Date,User
2020-03-02,ronda
2020-03-03,dan
2020-03-04,
2020-03-05,dan
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
				t.Errorf("\n--- EXPECTED ---\n%v\n--- RECEIVED ---\n%v", exp, s)
			}
		})
	}
}

func TestLoadOverrides(t *testing.T) {
	for _, tc := range []struct {
		description string
		in          string
		expected    map[string]string
	}{{
		description: "overrides parse ok from two fields",
		in: `
Date,User
2020-03-03,dan
2020-04-01,jane`,
		expected: map[string]string{
			"2020-03-03": "dan",
			"2020-04-01": "jane",
		},
	}, {
		description: "overrides are allowed to have extra fields for comments etc",
		in: `
Date,User
2020-03-03,dan,Swapping with Ronda on 2020-03-05
2020-03-05,ronda,Swapping with Dan on 2020-03-03
2020-04-01,jane`,
		expected: map[string]string{
			"2020-03-03": "dan",
			"2020-03-05": "ronda",
			"2020-04-01": "jane",
		},
	}, {
		description: "overrides are allowed to have blank users to indicate noone should buildcop that day",
		in: `
Date,User
2020-12-25,,Happy holidays!
		`,
		expected: map[string]string{
			"2020-12-25": "",
		},
	}} {
		t.Run(tc.description, func(t *testing.T) {
			r := strings.NewReader(strings.TrimSpace(tc.in))
			out, err := loadOverrides(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out) != len(tc.expected) {
				t.Fatalf("expected %d records but received %d", len(tc.expected), len(out))
			}
			for date, name := range out {
				if tc.expected[date] != name {
					t.Fatalf("expected %s:%s but received %s:%s", date, tc.expected[date], date, out[date])
				}
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
