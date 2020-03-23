package main

import (
	"encoding/csv"
	"os"
	"time"

	"k8s.io/kubernetes/pkg/kubelet/kubeletconfig/util/log"
)

// The Rotation object which knows how to dynamically find the correct build cop from
// the file it is initialized with.
type Rotation struct {
	file string
}

// NewRotation returns a new Rotation object which will read from file.
func NewRotation(file string) Rotation {
	return Rotation{file: file}
}

// GetBuildCop returns the name of the build cop for the requested time
func (r Rotation) GetBuildCop(t time.Time) string {
	tf := t.Format("2006-01-02") // Mon Jan 2 15:04:05 MST 2006
	rotation, err := readRotation(r.file)
	if err != nil {
		log.Errorf("Could not read from build cop rotation file %s: %v", r.file, err)
		return "nobody"
	}
	b, ok := rotation[tf]
	if !ok {
		log.Errorf("Couldn't find anyone in rotation %s for time %s", r.file, tf)
		return "nobody"
	}
	return b
}

func readRotation(filename string) (map[string]string, error) {
	rotation := map[string]string{}
	// Open CSV file
	f, err := os.Open(filename)
	if err != nil {
		return rotation, err
	}
	defer f.Close()

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
