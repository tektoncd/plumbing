package entry

import (
	"fmt"
	"testing"
)

func TestParseVersion(t *testing.T) {
	for _, tc := range []struct {
		from     string
		expected Version
	}{{
		from:     "0.1",
		expected: Version{0, 1},
	}, {
		from:     "0.8",
		expected: Version{0, 8},
	}, {
		from:     "9.0",
		expected: Version{9, 0},
	}, {
		from:     "3.14159",
		expected: Version{3, 14159},
	}, {
		from:     "780001.2",
		expected: Version{780001, 2},
	}, {
		from:     "00001.00004",
		expected: Version{1, 4},
	}} {
		t.Run(tc.from, func(t *testing.T) {
			parsed, err := ParseVersion(tc.from)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !parsed.Eq(tc.expected) {
				t.Fatalf("expected %s received %s", tc.expected, parsed)
			}
		})
	}
}

func TestParseVersionErrors(t *testing.T) {
	for _, tc := range []struct {
		from string
	}{{
		from: "abc0.1",
	}, {
		from: "0.abc8",
	}, {
		from: "_9.0",
	}, {
		from: "3.141_59",
	}, {
		from: "78,0001.2",
	}, {
		from: "00001.00,004",
	}, {
		from: "    1.9",
	}, {
		from: "1.9    ",
	}, {
		from: "this is not a version",
	}, {
		from: "1",
	}, {
		from: "1.",
	}, {
		from: ".1",
	}, {
		from: "1.1.1",
	}, {
		from: "o123.123",
	}, {
		from: "b1001.101",
	}, {
		from: "0x14.9",
	}} {
		t.Run(tc.from, func(t *testing.T) {
			_, err := ParseVersion(tc.from)
			if err == nil {
				t.Fatalf("expected error but %q appears to have been parsed as valid", tc.from)
			}
		})
	}
}

func TestString(t *testing.T) {
	for _, tc := range []struct {
		expected string
		from     Version
	}{{
		expected: "0.1",
		from:     Version{0, 1},
	}, {
		expected: "0.8",
		from:     Version{0, 8},
	}, {
		expected: "9.0",
		from:     Version{9, 0},
	}, {
		expected: "3.14159",
		from:     Version{3, 14159},
	}, {
		expected: "780001.2",
		from:     Version{780001, 2},
	}, {
		expected: "1.4",
		from:     Version{1, 4},
	}} {
		t.Run(tc.expected, func(t *testing.T) {
			str := tc.from.String()
			if str != tc.expected {
				t.Fatalf("expected %s received %s", tc.expected, str)
			}
		})
	}
}

func TestBumpMinor(t *testing.T) {
	for _, tc := range []struct {
		from     string
		expected Version
	}{{
		from:     "0.1",
		expected: Version{0, 2},
	}, {
		from:     "0.8",
		expected: Version{0, 9},
	}, {
		from:     "9.0",
		expected: Version{9, 1},
	}, {
		from:     "3.14159",
		expected: Version{3, 14160},
	}, {
		from:     "780001.2",
		expected: Version{780001, 3},
	}, {
		from:     "00001.00004",
		expected: Version{1, 5},
	}} {
		t.Run(tc.from, func(t *testing.T) {
			parsed, err := ParseVersion(tc.from)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			bumped := parsed.BumpMinor()
			if !bumped.Eq(tc.expected) {
				t.Fatalf("expected %s received %s", tc.expected, parsed)
			}
		})
	}
}

func TestGt(t *testing.T) {
	for _, tc := range []struct {
		upper    Version
		lower    Version
		expected bool
	}{{
		upper:    Version{0, 2},
		lower:    Version{0, 1},
		expected: true,
	}, {
		upper:    Version{0, 9},
		lower:    Version{0, 8},
		expected: true,
	}, {
		upper:    Version{9, 1},
		lower:    Version{6, 0},
		expected: true,
	}, {
		upper:    Version{3, 14160},
		lower:    Version{3, 1},
		expected: true,
	}, {
		upper:    Version{780001, 3},
		lower:    Version{780000, 3},
		expected: true,
	}, {
		lower:    Version{0, 2},
		upper:    Version{0, 1},
		expected: false,
	}, {
		lower:    Version{0, 9},
		upper:    Version{0, 8},
		expected: false,
	}, {
		lower:    Version{9, 1},
		upper:    Version{6, 0},
		expected: false,
	}, {
		lower:    Version{3, 14160},
		upper:    Version{3, 1},
		expected: false,
	}, {
		lower:    Version{780001, 3},
		upper:    Version{780000, 3},
		expected: false,
	}} {
		t.Run(fmt.Sprintf("%s > %s", tc.upper, tc.lower), func(t *testing.T) {
			if tc.upper.Gt(tc.lower) != tc.expected {
				t.Fatalf("expected %s > %s to be %t", tc.upper, tc.lower, tc.expected)
			}
		})
	}
}

func TestLt(t *testing.T) {
	for _, tc := range []struct {
		upper    Version
		lower    Version
		expected bool
	}{{
		upper:    Version{0, 2},
		lower:    Version{0, 1},
		expected: false,
	}, {
		upper:    Version{0, 9},
		lower:    Version{0, 8},
		expected: false,
	}, {
		upper:    Version{9, 1},
		lower:    Version{6, 0},
		expected: false,
	}, {
		upper:    Version{3, 14160},
		lower:    Version{3, 1},
		expected: false,
	}, {
		upper:    Version{780001, 3},
		lower:    Version{780000, 3},
		expected: false,
	}, {
		lower:    Version{0, 2},
		upper:    Version{0, 1},
		expected: true,
	}, {
		lower:    Version{0, 9},
		upper:    Version{0, 8},
		expected: true,
	}, {
		lower:    Version{9, 1},
		upper:    Version{6, 0},
		expected: true,
	}, {
		lower:    Version{3, 14160},
		upper:    Version{3, 1},
		expected: true,
	}, {
		lower:    Version{780001, 3},
		upper:    Version{780000, 3},
		expected: true,
	}} {
		t.Run(fmt.Sprintf("%s < %s", tc.lower, tc.upper), func(t *testing.T) {
			if tc.upper.Lt(tc.lower) != tc.expected {
				t.Fatalf("expected %s > %s to be %t", tc.upper, tc.lower, tc.expected)
			}
		})
	}
}
