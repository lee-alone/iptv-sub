package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// CreateTempDir creates a temporary directory for testing
func CreateTempDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "iptv-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tmpDir
}

// CleanupTempDir removes a temporary directory
func CleanupTempDir(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Logf("Failed to cleanup temp directory %s: %v", dir, err)
	}
}

// CreateTestFile creates a test file with the given content
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	return filePath
}

// ReadTestFile reads the content of a test file
func ReadTestFile(t *testing.T, filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", filePath, err)
	}
	return string(content)
}

// AssertEqual asserts that two values are equal
func AssertEqual(t *testing.T, expected, actual any, message string) {
	if expected != actual {
		t.Errorf("Assertion failed: %s\nExpected: %v\nActual: %v", message, expected, actual)
	}
}

// AssertNotEqual asserts that two values are not equal
func AssertNotEqual(t *testing.T, expected, actual any, message string) {
	if expected == actual {
		t.Errorf("Assertion failed: %s\nExpected not equal to: %v\nActual: %v", message, expected, actual)
	}
}

// AssertTrue asserts that a condition is true
func AssertTrue(t *testing.T, condition bool, message string) {
	if !condition {
		t.Errorf("Assertion failed: %s (expected true)", message)
	}
}

// AssertFalse asserts that a condition is false
func AssertFalse(t *testing.T, condition bool, message string) {
	if condition {
		t.Errorf("Assertion failed: %s (expected false)", message)
	}
}

// AssertNil asserts that a value is nil
func AssertNil(t *testing.T, value any, message string) {
	if value != nil {
		t.Errorf("Assertion failed: %s\nExpected nil, got: %v", message, value)
	}
}

// AssertNotNil asserts that a value is not nil
func AssertNotNil(t *testing.T, value any, message string) {
	if value == nil {
		t.Errorf("Assertion failed: %s (expected not nil)", message)
	}
}

// AssertError asserts that an error occurred
func AssertError(t *testing.T, err error, message string) {
	if err == nil {
		t.Errorf("Assertion failed: %s (expected error)", message)
	}
}

// AssertNoError asserts that no error occurred
func AssertNoError(t *testing.T, err error, message string) {
	if err != nil {
		t.Errorf("Assertion failed: %s\nError: %v", message, err)
	}
}

// AssertErrorMessage asserts that an error message contains a specific string
func AssertErrorMessage(t *testing.T, err error, expectedMsg string, message string) {
	if err == nil {
		t.Errorf("Assertion failed: %s (expected error)", message)
		return
	}
	if err.Error() != expectedMsg {
		t.Errorf("Assertion failed: %s\nExpected error message: %s\nActual: %s", message, expectedMsg, err.Error())
	}
}

// AssertSliceEqual asserts that two slices are equal
func AssertSliceEqual(t *testing.T, expected, actual []string, message string) {
	if len(expected) != len(actual) {
		t.Errorf("Assertion failed: %s\nExpected length: %d\nActual length: %d", message, len(expected), len(actual))
		return
	}
	for i, v := range expected {
		if v != actual[i] {
			t.Errorf("Assertion failed: %s\nAt index %d: expected %s, got %s", message, i, v, actual[i])
		}
	}
}

// AssertContains asserts that a slice contains a specific value
func AssertContains(t *testing.T, slice []string, value string, message string) {
	if !slices.Contains(slice, value) {
		t.Errorf("Assertion failed: %s\nExpected slice to contain: %s", message, value)
	}
}

// AssertNotContains asserts that a slice does not contain a specific value
func AssertNotContains(t *testing.T, slice []string, value string, message string) {
	if slices.Contains(slice, value) {
		t.Errorf("Assertion failed: %s\nExpected slice not to contain: %s", message, value)
	}
}

// AssertStringContains asserts that a string contains a substring
func AssertStringContains(t *testing.T, str, substr, message string) {
	if !contains(str, substr) {
		t.Errorf("Assertion failed: %s\nExpected string to contain: %s\nActual: %s", message, substr, str)
	}
}

// AssertStringNotContains asserts that a string does not contain a substring
func AssertStringNotContains(t *testing.T, str, substr, message string) {
	if contains(str, substr) {
		t.Errorf("Assertion failed: %s\nExpected string not to contain: %s\nActual: %s", message, substr, str)
	}
}

// contains checks if a string contains a substring
func contains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// AssertPanic asserts that a function panics
func AssertPanic(t *testing.T, fn func(), message string) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Assertion failed: %s (expected panic)", message)
		}
	}()
	fn()
}

// AssertNoPanic asserts that a function does not panic
func AssertNoPanic(t *testing.T, fn func(), message string) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Assertion failed: %s (unexpected panic: %v)", message, r)
		}
	}()
	fn()
}

// SkipIfShort skips the test if the -short flag is set
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// Logf logs a formatted message
func Logf(t *testing.T, format string, args ...any) {
	t.Logf(format, args...)
}

// Fatalf logs a formatted message and fails the test
func Fatalf(t *testing.T, format string, args ...any) {
	t.Fatalf(format, args...)
}

// Errorf logs a formatted error message
func Errorf(t *testing.T, format string, args ...any) {
	t.Errorf(format, args...)
}

// GetTestDataPath returns the path to the test data directory
func GetTestDataPath() string {
	return filepath.Join("tests", "testdata")
}

// GetTestDataFile returns the full path to a test data file
func GetTestDataFile(filename string) string {
	return filepath.Join(GetTestDataPath(), filename)
}

// FileExists checks if a file exists
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// DirExists checks if a directory exists
func DirExists(dirPath string) bool {
	info, err := os.Stat(dirPath)
	return err == nil && info.IsDir()
}

// CreateTestDir creates a test directory if it doesn't exist
func CreateTestDir(t *testing.T, dirPath string) {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory %s: %v", dirPath, err)
	}
}

// RemoveTestDir removes a test directory
func RemoveTestDir(t *testing.T, dirPath string) {
	if err := os.RemoveAll(dirPath); err != nil {
		t.Logf("Failed to remove test directory %s: %v", dirPath, err)
	}
}

// PrintTestInfo prints test information
func PrintTestInfo(t *testing.T, testName string) {
	fmt.Printf("\n=== Running test: %s ===\n", testName)
}

// PrintTestResult prints test result
func PrintTestResult(t *testing.T, testName string, passed bool) {
	status := "PASSED"
	if !passed {
		status = "FAILED"
	}
	fmt.Printf("=== Test %s: %s ===\n\n", testName, status)
}
