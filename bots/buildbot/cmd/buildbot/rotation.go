package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// The Rotation object which knows how to dynamically find the correct build cop from
// the file getter f it is initialized with.
type Rotation struct {
	f GetFile
}

// GetFile is the signature of a function that knows how to retrieve the bytes from a file
type GetFile func() (io.ReadCloser, error)

// FromURL is a GetFile that knows how to read a file from a url.
func FromURL(url string) GetFile {
	return func() (io.ReadCloser, error) {
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("Could not open url %s: %v", url, err)
		}
		return resp.Body, nil
	}
}

// NewRotation returns a new Rotation object which uses f to retrieve the rotation
// file as needed.
func NewRotation(f GetFile) Rotation {
	return Rotation{f: f}
}

// GetBuildCop returns the name of the build cop for the requested time
func (r Rotation) GetBuildCop(t time.Time) string {
	tf := t.Format("2006-01-02") // Mon Jan 2 15:04:05 MST 2006
	f, err := r.f()
	if err != nil {
		log.Printf("Could not read from build cop rotation: %v", err)
		return "nobody"
	}
	defer f.Close()
	rotation, err := parseRotation(f)

	if err != nil {
		log.Printf("Could not read rotation from build cop rotation: %v", err)
		return "nobody"
	}
	b, ok := rotation[tf]
	if !ok {
		log.Printf("Couldn't find anyone in rotation for time %s", tf)
		return "nobody"
	}
	return b
}

func parseRotation(f io.Reader) (map[string]string, error) {
	rotation := map[string]string{}
	// Read File into a Variable
	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return rotation, err
	}

	for i, line := range lines {
		if i == 0 {
			continue
		}
		rotation[line[0]] = line[1]
	}
	return rotation, nil
}
