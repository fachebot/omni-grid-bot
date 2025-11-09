package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

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

type Config struct {
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
		c.AppName = "QuantGrid"
	}

	return &c, nil
}
