package consumption

import (
	"path/filepath"
	"testing"
)

var (
	testRootPath = "/mnt/rootDir/"
)

func TestParseTargetRootSubdir(t *testing.T) {

	path := filepath.Join(testRootPath, "testDir")
	target, err := parseTargetOnRootPath(testRootPath, path, false)
	if err != nil {
		t.Fatal(err)
	}
	if target != path {
		t.Errorf("parsed path incorrect, %s (expected) != %s (actual)", path, target)
	}
}

func TestParseTargetNotInRoot(t *testing.T) {
	path := filepath.Join(testRootPath, "../testDir")
	_, err := parseTargetOnRootPath(testRootPath, path, false)
	if err == nil {
		t.Errorf("did not throw an ErrNotInRoot error for path %s and root %s", path, testRootPath)
	}
}

func TestParseTargetNotInRootAllowAnywhere(t *testing.T) {
	path := filepath.Join(testRootPath, "../testDir")
	_, err := parseTargetOnRootPath(testRootPath, path, true)
	if err != nil {
		t.Error(err)
	}
}

func TestParseTargetRelPath(t *testing.T) {

	path := "testDir"
	target, err := parseTargetOnRootPath(testRootPath, path, false)
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(testRootPath, path)
	if target != expected {
		t.Errorf("parsed path incorrect, %s (expected) != %s (actual)", expected, target)
	}
}

func TestParseTargetDotPath(t *testing.T) {

	path := "./testDir"
	target, err := parseTargetOnRootPath(testRootPath, path, false)
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(testRootPath, path)
	if target != expected {
		t.Errorf("parsed path incorrect, %s (expected) != %s (actual)", expected, target)
	}
}
