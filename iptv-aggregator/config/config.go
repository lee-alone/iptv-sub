package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config 系统配置结构体
type Config struct {
	// 服务器配置
	Port int    `json:"port"`
	Host string `json:"host"`

	// 请求配置
	RequestTimeout    time.Duration `json:"request_timeout"`
	StreamTestTimeout time.Duration `json:"stream_test_timeout"`
	MaxTestWorkers    int           `json:"max_test_workers"`

	// 调度配置
	UpdateInterval   time.Duration `json:"update_interval"`
	TestInterval     time.Duration `json:"test_interval"`
	EnableStreamTest bool          `json:"enable_stream_test"`
	TestAllSources   bool          `json:"test_all_sources"`

	// 聚合配置
	MatchBy             string  `json:"match_by"` // name, tvg_id, both
	SimilarityThreshold float64 `json:"similarity_threshold"`

	// 深度检查配置
	DeepCheck     bool          `json:"deep_check"`
	LoopChecks    int           `json:"loop_checks"`
	LoopInterval  time.Duration `json:"loop_interval"`
	SegmentWindow int           `json:"segment_window"`

	// 数据配置
	DataDir string `json:"data_dir"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Port:                8080,
		Host:                "0.0.0.0",
		RequestTimeout:      30 * time.Second,
		StreamTestTimeout:   5 * time.Second,
		MaxTestWorkers:      10,
		UpdateInterval:      24 * time.Hour,
		TestInterval:        24 * time.Hour,
		EnableStreamTest:    true,
		TestAllSources:      false,
		MatchBy:             "name",
		SimilarityThreshold: 0.85,
		DeepCheck:           true,
		LoopChecks:          3,
		LoopInterval:        4 * time.Second,
		SegmentWindow:       5,
		DataDir:             "data",
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(filePath string) (*Config, error) {
	// 使用默认配置
	cfg := DefaultConfig()

	// 如果文件不存在，创建默认配置文件
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := SaveConfig(filePath, cfg); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析 JSON
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(filePath string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 原子写入：先写临时文件，再重命名
	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := os.Rename(tmpFile, filePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	return nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.MaxTestWorkers < 1 || c.MaxTestWorkers > 100 {
		return fmt.Errorf("invalid max_test_workers: %d", c.MaxTestWorkers)
	}

	if c.SimilarityThreshold < 0 || c.SimilarityThreshold > 1 {
		return fmt.Errorf("invalid similarity_threshold: %f", c.SimilarityThreshold)
	}

	if c.MatchBy != "name" && c.MatchBy != "tvg_id" && c.MatchBy != "both" {
		return fmt.Errorf("invalid match_by: %s", c.MatchBy)
	}

	return nil
}
