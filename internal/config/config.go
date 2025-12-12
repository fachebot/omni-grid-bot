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
	Log         Log         `yaml:"Log"`
	AppName     string      `yaml:"AppName"`
	Sock5Proxy  Sock5Proxy  `yaml:"Sock5Proxy"`
	TelegramBot TelegramBot `yaml:"TelegramBot"`
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

	return &c, nil
}
