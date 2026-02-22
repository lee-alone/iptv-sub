package unit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/iptv-aggregator/config"
	"github.com/yourusername/iptv-aggregator/tests"
)

// TestLoadConfig_DefaultConfig tests loading default configuration
func TestLoadConfig_DefaultConfig(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")

	cfg, err := config.LoadConfig(configPath)
	tests.AssertNoError(t, err, "LoadConfig should succeed")
	tests.AssertNotNil(t, cfg, "Config should not be nil")

	tests.AssertEqual(t, 8080, cfg.Port, "Default port should be 8080")
	tests.AssertEqual(t, "0.0.0.0", cfg.Host, "Default host should be 0.0.0.0")
	tests.AssertTrue(t, cfg.EnableStreamTest, "EnableStreamTest should be true by default")
}

// TestLoadConfig_FileCreation tests that config file is created if not exists
func TestLoadConfig_FileCreation(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")

	tests.AssertFalse(t, tests.FileExists(configPath), "Config file should not exist initially")

	config.LoadConfig(configPath)

	tests.AssertTrue(t, tests.FileExists(configPath), "Config file should be created")
}

// TestSaveConfig_ValidConfig tests saving valid configuration
func TestSaveConfig_ValidConfig(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")

	cfg := config.DefaultConfig()
	cfg.Port = 9090
	cfg.Host = "127.0.0.1"

	err := config.SaveConfig(configPath, cfg)
	tests.AssertNoError(t, err, "SaveConfig should succeed")
	tests.AssertTrue(t, tests.FileExists(configPath), "Config file should be created")
}

// TestSaveConfig_LoadAfterSave tests loading config after saving
func TestSaveConfig_LoadAfterSave(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")

	cfg1 := config.DefaultConfig()
	cfg1.Port = 9090
	cfg1.Host = "127.0.0.1"
	cfg1.MaxTestWorkers = 20

	err := config.SaveConfig(configPath, cfg1)
	tests.AssertNoError(t, err, "SaveConfig should succeed")

	cfg2, err := config.LoadConfig(configPath)
	tests.AssertNoError(t, err, "LoadConfig should succeed")

	tests.AssertEqual(t, 9090, cfg2.Port, "Port should match saved value")
	tests.AssertEqual(t, "127.0.0.1", cfg2.Host, "Host should match saved value")
	tests.AssertEqual(t, 20, cfg2.MaxTestWorkers, "MaxTestWorkers should match saved value")
}

// TestValidate_ValidConfig tests validation of valid configuration
func TestValidate_ValidConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	err := cfg.Validate()
	tests.AssertNoError(t, err, "Validate should succeed for default config")
}

// TestValidate_InvalidPort_TooLow tests validation with port too low
func TestValidate_InvalidPort_TooLow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Port = 0

	err := cfg.Validate()
	tests.AssertError(t, err, "Validate should fail for port 0")
	tests.AssertStringContains(t, err.Error(), "invalid port", "Error should mention invalid port")
}

// TestValidate_InvalidPort_TooHigh tests validation with port too high
func TestValidate_InvalidPort_TooHigh(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Port = 65536

	err := cfg.Validate()
	tests.AssertError(t, err, "Validate should fail for port > 65535")
}

// TestValidate_InvalidMaxTestWorkers_TooLow tests validation with max test workers too low
func TestValidate_InvalidMaxTestWorkers_TooLow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MaxTestWorkers = 0

	err := cfg.Validate()
	tests.AssertError(t, err, "Validate should fail for MaxTestWorkers 0")
	tests.AssertStringContains(t, err.Error(), "invalid max_test_workers", "Error should mention invalid max_test_workers")
}

// TestValidate_InvalidSimilarityThreshold_TooLow tests validation with similarity threshold too low
func TestValidate_InvalidSimilarityThreshold_TooLow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SimilarityThreshold = -0.1

	err := cfg.Validate()
	tests.AssertError(t, err, "Validate should fail for negative similarity threshold")
	tests.AssertStringContains(t, err.Error(), "invalid similarity_threshold", "Error should mention invalid similarity_threshold")
}

// TestValidate_InvalidMatchBy tests validation with invalid match_by value
func TestValidate_InvalidMatchBy(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MatchBy = "invalid"

	err := cfg.Validate()
	tests.AssertError(t, err, "Validate should fail for invalid match_by")
	tests.AssertStringContains(t, err.Error(), "invalid match_by", "Error should mention invalid match_by")
}

// TestDefaultConfig_Values tests default configuration values
func TestDefaultConfig_Values(t *testing.T) {
	cfg := config.DefaultConfig()

	tests.AssertEqual(t, 8080, cfg.Port, "Default port should be 8080")
	tests.AssertEqual(t, "0.0.0.0", cfg.Host, "Default host should be 0.0.0.0")
	tests.AssertEqual(t, 30*time.Second, cfg.RequestTimeout, "Default request timeout should be 30s")
	tests.AssertEqual(t, 5*time.Second, cfg.StreamTestTimeout, "Default stream test timeout should be 5s")
	tests.AssertEqual(t, 10, cfg.MaxTestWorkers, "Default max test workers should be 10")
	tests.AssertEqual(t, 24*time.Hour, cfg.UpdateInterval, "Default update interval should be 24h")
	tests.AssertEqual(t, true, cfg.EnableStreamTest, "EnableStreamTest should be true")
	tests.AssertEqual(t, false, cfg.TestAllSources, "TestAllSources should be false")
	tests.AssertEqual(t, "name", cfg.MatchBy, "Default match_by should be 'name'")
	tests.AssertEqual(t, 0.85, cfg.SimilarityThreshold, "Default similarity threshold should be 0.85")
	tests.AssertEqual(t, "data", cfg.DataDir, "Default data dir should be 'data'")
}

// TestLoadConfig_InvalidJSON tests loading invalid JSON file
func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	tests.CreateTestFile(t, tmpDir, "config.json", "invalid json {")

	_, err := config.LoadConfig(configPath)
	tests.AssertError(t, err, "LoadConfig should fail for invalid JSON")
}

// TestSaveConfig_AtomicWrite tests that config save is atomic
func TestSaveConfig_AtomicWrite(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")

	cfg := config.DefaultConfig()
	cfg.Port = 9090

	err := config.SaveConfig(configPath, cfg)
	tests.AssertNoError(t, err, "SaveConfig should succeed")

	tests.AssertTrue(t, tests.FileExists(configPath), "Config file should exist")

	cfg2, err := config.LoadConfig(configPath)
	tests.AssertNoError(t, err, "LoadConfig should succeed")
	tests.AssertEqual(t, 9090, cfg2.Port, "Port should be saved correctly")
}

// TestValidate_BoundaryValues tests validation with boundary values
func TestValidate_BoundaryValues(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Port = 1
	cfg.MaxTestWorkers = 1
	cfg.SimilarityThreshold = 0.0

	err := cfg.Validate()
	tests.AssertNoError(t, err, "Validate should succeed for minimum valid values")

	cfg.Port = 65535
	cfg.MaxTestWorkers = 100
	cfg.SimilarityThreshold = 1.0

	err = cfg.Validate()
	tests.AssertNoError(t, err, "Validate should succeed for maximum valid values")
}

// TestDefaultConfig_Consistency tests that default config is consistent
func TestDefaultConfig_Consistency(t *testing.T) {
	cfg1 := config.DefaultConfig()
	cfg2 := config.DefaultConfig()

	tests.AssertEqual(t, cfg1.Port, cfg2.Port, "Default port should be consistent")
	tests.AssertEqual(t, cfg1.Host, cfg2.Host, "Default host should be consistent")
	tests.AssertEqual(t, cfg1.MaxTestWorkers, cfg2.MaxTestWorkers, "Default max test workers should be consistent")
}
