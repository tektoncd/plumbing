package tep_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
)

func TestGetTEPsWithStatus(t *testing.T) {
	testCases := []struct {
		name        string
		status      tep.Status
		input       map[string]tep.TEPInfo
		expectedIDs []string
	}{
		{
			name:   "none",
			status: tep.ProposedStatus,
			input: map[string]tep.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP",
					Status:       tep.ImplementableStatus,
					Filename:     "1234.md",
					LastModified: time.Time{},
				},
			},
		},
		{
			name:   "one match",
			status: tep.ProposedStatus,
			input: map[string]tep.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP",
					Status:       tep.ImplementableStatus,
					Filename:     "1234.md",
					LastModified: time.Time{},
				},
				"4321": {
					ID:           "4321",
					Title:        "Some Other TEP",
					Status:       tep.ProposedStatus,
					Filename:     "4321.md",
					LastModified: time.Time{},
				},
			},
			expectedIDs: []string{"4321"},
		},
		{
			name:   "two match",
			status: tep.ProposedStatus,
			input: map[string]tep.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP",
					Status:       tep.ImplementableStatus,
					Filename:     "1234.md",
					LastModified: time.Time{},
				},
				"4321": {
					ID:           "4321",
					Title:        "Some Other TEP",
					Status:       tep.ProposedStatus,
					Filename:     "4321.md",
					LastModified: time.Time{},
				},
				"5678": {
					ID:           "5678",
					Title:        "A Third TEP",
					Status:       tep.ImplementedStatus,
					Filename:     "5678.md",
					LastModified: time.Time{},
				},
				"8765": {
					ID:           "8765",
					Title:        "A Fourth TEP",
					Status:       tep.ProposedStatus,
					Filename:     "8765.md",
					LastModified: time.Time{},
				},
			},
			expectedIDs: []string{"4321", "8765"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			found := tep.GetTEPsWithStatus(tc.input, tc.status)
			var foundIDs []string
			for k := range found {
				foundIDs = append(foundIDs, k)
			}
			assert.ElementsMatch(t, tc.expectedIDs, foundIDs)
		})
	}
}
