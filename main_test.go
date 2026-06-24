package main

import (
	"testing"
)

func TestFindAbsPath(t *testing.T) {
	testCases := []struct {
		name     string
		fileName string
		expected string
	}{
		{"file 1", "file1.txt", "/data/coding/Go/tui/list2/files/file1.txt"},
		{"file 2", "file2.txt", "/data/coding/Go/tui/list2/files/file2.txt"},
		{"file 10", "file10.txt", "/data/coding/Go/tui/list2/files/file10.txt"},
		{"file random", "random.txt", "/data/coding/Go/tui/list2/files/random.txt"},
		{"files Dir", "", "/data/coding/Go/tui/list2/files"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findAbsPath(baseDir, tc.fileName)

			if result != tc.expected {
				t.Errorf("Expected %s, but got %s", tc.expected, result)
			}
		})
	}
}

func TestCopyToClipboard(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected error
	}{
		{"text 1", "text1", nil},
		{"text 2", "text2", nil},
		{"file path1", "/data/coding/Go/tui/list2/files/file1.txt", nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := copyToClipboard(tc.text)

			if err != tc.expected {
				t.Errorf("Expected %v, but got %s", nil, err.Error())
			}
		})
	}
}

