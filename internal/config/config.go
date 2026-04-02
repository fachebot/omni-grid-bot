package config

import (
	"os"
	"slices"

	"gopkg.in/yaml.v2"
)

type Log struct {
	Level string `yaml:"Level"`
}

type Sock5Proxy struct {
	Host   string `yaml:"Host"`
	Port   int32  `yaml:"Port"`
	Enable bool   `yaml:"Enable"`
}

type LighterRateLimit struct {
	RequestsPerMinute int `yaml:"RequestsPerMinute"` // 默认60
}

type VariationalRateLimit struct {
	RequestsPerSecond float64 `yaml:"RequestsPerSecond"` // 默认1.0
	Burst             int     `yaml:"Burst"`             // 默认1
}

type TelegramBot struct {
	Debug        bool    `yaml:"Debug"`
	ApiToken     string  `yaml:"ApiToken"`
	WhiteList    []int64 `yaml:"WhiteList"`
	NotifyChatId int64   `yaml:"NotifyChatId"`
}

func (c *TelegramBot) IsWhiteListUser(userId int64) bool {
	if len(c.WhiteList) == 0 {
		return true
	}
	return slices.Contains(c.WhiteList, userId)
}

type Config struct {
	Log                  Log                  `yaml:"Log"`
	AppName              string               `yaml:"AppName"`
	Sock5Proxy           Sock5Proxy           `yaml:"Sock5Proxy"`
	TelegramBot          TelegramBot          `yaml:"TelegramBot"`
	LighterRateLimit     LighterRateLimit     `yaml:"LighterRateLimit"`
	VariationalRateLimit VariationalRateLimit `yaml:"VariationalRateLimit"`
}

func LoadFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var c Config
	err = yaml.Unmarshal([]byte(data), &c)
	if err != nil {
		return nil, err
	}

	if c.AppName == "" {
		c.AppName = "OmniGrid"
	}

	if c.Log.Level == "" {
		c.Log.Level = "info"
	}

	if c.LighterRateLimit.RequestsPerMinute == 0 {
		c.LighterRateLimit.RequestsPerMinute = 60
	}

	if c.VariationalRateLimit.RequestsPerSecond == 0 {
		c.VariationalRateLimit.RequestsPerSecond = 1.0
	}

	if c.VariationalRateLimit.Burst == 0 {
		c.VariationalRateLimit.Burst = 1
	}

	return &c, nil
}
