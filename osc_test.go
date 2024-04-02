package main

import (
	"testing"
)

func assertGetTimeSkip(message string, expected int64, t *testing.T) {
	result := getTimeSkip(message)
	if result != expected {
		t.Fatalf("Error on %s. Expected %d, got %d", message, expected, result)
	}
}

func TestGetTime(t *testing.T) {
	assertGetTimeSkip("/in", 0, t)
	assertGetTimeSkip("/dummy", 0, t)
	assertGetTimeSkip("/inc", 60, t)
	assertGetTimeSkip("/inc10", 600, t)
	assertGetTimeSkip("/chronono/inc10", 600, t)
	assertGetTimeSkip("/inc10m", 600, t)
	assertGetTimeSkip("/inc1", 60, t)
	assertGetTimeSkip("/inc1s", 1, t)
	assertGetTimeSkip("/incs", 1, t)
	assertGetTimeSkip("/inc74s", 74, t)
	assertGetTimeSkip("/dec", -60, t)
	assertGetTimeSkip("/dec10", -600, t)
	assertGetTimeSkip("/decs", -1, t)
	assertGetTimeSkip("/dec1", -60, t)
}
