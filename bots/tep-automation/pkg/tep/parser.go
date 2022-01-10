package tep

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
)

var (
	// TEPsInReadme matches rows in the table in https://github.com/tektoncd/community/blob/main/teps/README.md
	TEPsInReadme = regexp.MustCompile(`\|\[TEP-(\d+)]\((.*?\.md)\) \| (.*?) \| (.*?) \| (\d\d\d\d-\d\d-\d\d) \|`)
	// NotifierActionRegex is used to detect whether a comment is referring to transitioning to implementing or to implemented.
	NotifierActionRegex = regexp.MustCompile(`<!-- TEP Notifier Action: (\w+) -->`)
	// IDRegex is used to parse out "TEP-1234" from PR bodies and titles.
	IDRegex = regexp.MustCompile(`TEP-(\d+)`)
	// URLRegex is used to parse out TEP URLs, like https://github.com/tektoncd/community/blob/main/teps/0002-custom-tasks.md, from
	// PR bodies. We ignore the branch when looking for matches.
	URLRegex = regexp.MustCompile(`https://github\.com/tektoncd/community/blob/.*?/teps/(\d+)-.*?\.md`)
	// CommentAndStatusRegex is used to detect whether a comment already is being used for the reminder message for
	// a particular TEP and status.
	CommentAndStatusRegex = regexp.MustCompile(`<!-- TEP update: TEP-(\d+) status: (\w+) -->`)
	// TrackingIssueTEPPRsRegex parses out references to community repo TEP PRs from an issue's body.
	TrackingIssueTEPPRsRegex = regexp.MustCompile(`<!-- TEP PR: (\d+) -->`)
	// TrackingIssueImplementationPRsRegex parses out references to PRs implementing a TEP from an issue's body.
	TrackingIssueImplementationPRsRegex = regexp.MustCompile(`<!-- Implementation PR: repo: (.*?) number: (\d+) -->`)
)

// ExtractTEPsFromReadme takes the body of https://github.com/tektoncd/community/blob/main/teps/README.md and extracts a
// map of all TEP IDs (i.e., "1234" for TEP-1234) and their statuses.
func ExtractTEPsFromReadme(readmeBody string) (map[string]TEPInfo, error) {
	teps := make(map[string]TEPInfo)

	for _, m := range TEPsInReadme.FindAllStringSubmatch(readmeBody, -1) {
		if len(m) > 5 {
			// TODO(abayer): For some reason, I can't ever get time.Parse to handle a format of "2021-12-20" so let's just pad it and do it as RFC3339.
			lastMod, err := time.Parse(time.RFC3339, fmt.Sprintf("%sT00:00:00Z", m[5]))
			if err != nil {
				return nil, err
			}

			if !IsValidStatus(m[4]) {
				return nil, fmt.Errorf("%s is not a valid status", m[4])
			}
			teps[m[1]] = TEPInfo{
				ID:           m[1],
				Title:        m[3],
				Status:       Status(m[4]),
				Filename:     m[2],
				LastModified: lastMod,
			}
		}
	}

	return teps, nil
}

// GetTEPCommentDetails looks at a PR comment and extracts any TEP IDs and their status in the comment and whether this
// comment is for transitioning TEPs to `implementing` or to `implemented`
func GetTEPCommentDetails(comment string) (map[string]string, bool) {
	teps := make(map[string]string)

	toImplemented := false
	notifierMatch := NotifierActionRegex.FindStringSubmatch(comment)
	if len(notifierMatch) > 1 && notifierMatch[1] == string(ImplementedStatus) {
		toImplemented = true
	}

	for _, m := range CommentAndStatusRegex.FindAllStringSubmatch(comment, -1) {
		if len(m) > 2 {
			teps[m[1]] = m[2]
		}
	}

	return teps, toImplemented
}

// GetTEPIDsFromPR extracts all TEP IDs and URLs in the given PR title and body
func GetTEPIDsFromPR(prTitle, prBody string) []string {
	var tepIDs []string

	// Find "TEP-1234" in PR title
	for _, m := range IDRegex.FindAllStringSubmatch(prTitle, -1) {
		if len(m) > 1 {
			tepIDs = append(tepIDs, m[1])
		}
	}
	// Find TEP URLs in title
	for _, m := range URLRegex.FindAllStringSubmatch(prTitle, -1) {
		if len(m) > 1 {
			tepIDs = append(tepIDs, m[1])
		}
	}

	// Find "TEP-1234" in PR body
	for _, m := range IDRegex.FindAllStringSubmatch(prBody, -1) {
		if len(m) > 1 {
			tepIDs = append(tepIDs, m[1])
		}
	}
	// Find TEP URLs in body
	for _, m := range URLRegex.FindAllStringSubmatch(prBody, -1) {
		if len(m) > 1 {
			tepIDs = append(tepIDs, m[1])
		}
	}

	return tepIDs
}

// TEPInfoFromMarkdown parses a TEP markdown file to get its ID, title, status, last modified time, and authors
func TEPInfoFromMarkdown(id string, filename string, contents string) (TEPInfo, error) {
	info := TEPInfo{
		ID:       id,
		Filename: filename,
		Authors:  []string{},
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
		),
	)
	var buf bytes.Buffer
	parserCtx := parser.NewContext()
	if err := markdown.Convert([]byte(contents), &buf, parser.WithContext(parserCtx)); err != nil {
		return info, err
	}
	md := meta.Get(parserCtx)

	title, ok := md["title"].(string)
	if !ok {
		return info, fmt.Errorf("metadata for title '%s' is not a string", md["title"])
	}
	info.Title = title

	rawStatus, ok := md["status"].(string)
	if !ok {
		return info, fmt.Errorf("metadata for status '%s' is not a string", md["status"])
	}
	if !IsValidStatus(rawStatus) {
		return info, fmt.Errorf("metadata for status '%s' is not a valid status", rawStatus)
	}
	info.Status = Status(rawStatus)

	rawLastMod, ok := md["last-updated"].(string)
	if !ok {
		return info, fmt.Errorf("metadata for last-updated '%s' is not a string", md["last-updated"])
	}
	lastMod, err := time.Parse(time.RFC3339, fmt.Sprintf("%sT00:00:00Z", rawLastMod))
	if err != nil {
		return info, errors.Wrapf(err, "couldn't parse last-updated '%s'", rawLastMod)
	}
	info.LastModified = lastMod

	rawAuthors, ok := md["authors"].([]interface{})
	if !ok {
		return info, fmt.Errorf("metadata for authors '%s' is not a slice", md["authors"])
	}
	for _, ra := range rawAuthors {
		a, ok := ra.(string)
		if !ok {
			return info, fmt.Errorf("metadata for individual author '%s' is not a string", ra)
		}
		info.Authors = append(info.Authors, strings.TrimPrefix(a, "@"))
	}

	return info, nil
}

// PRsForTrackingIssue parses the body of a tracking issue to find all TEP PRs and implementation PRs within the metadata
// in the body, and returns them.
func PRsForTrackingIssue(body string) ([]int, []ImplementationPR, error) {
	var tepPRIDs []int
	var implPRs []ImplementationPR

	for _, m := range TrackingIssueTEPPRsRegex.FindAllStringSubmatch(body, -1) {
		if len(m) > 1 {
			id, err := strconv.Atoi(m[1])
			if err != nil {
				return nil, nil, err
			}
			tepPRIDs = append(tepPRIDs, id)
		}
	}

	for _, m := range TrackingIssueImplementationPRsRegex.FindAllStringSubmatch(body, -1) {
		if len(m) > 1 {
			id, err := strconv.Atoi(m[2])
			if err != nil {
				return nil, nil, err
			}
			implPRs = append(implPRs, ImplementationPR{
				Repo:   m[1],
				Number: id,
			})
		}
	}

	return tepPRIDs, implPRs, nil
}
