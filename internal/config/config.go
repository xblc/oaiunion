package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config 是整個服務的設定結構
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
	Routing   RoutingConfig    `yaml:"routing"`
}

// ServerConfig 定義了服務監聽的設定
type ServerConfig struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	APIKey string `yaml:"api_key"`
}

// EndpointConfig 定義了後端 API 端點的設定
type EndpointConfig struct {
	Name    string `yaml:"name"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Weight  int    `yaml:"weight"`
}

// RoutingConfig 定義了模型路由和合併策略
type RoutingConfig struct {
	Mode           string              `yaml:"mode"`
	ModelOverrides map[string][]string `yaml:"model_overrides"`
}

// LoadConfig 從指定的路徑讀取並解析 YAML 設定檔
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
