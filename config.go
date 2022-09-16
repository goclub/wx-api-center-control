package main

import (
	"fmt"
	xerr "github.com/goclub/error"
	"strings"
)

type Config struct {
	HttpPort string      `yaml:"http_port"`
	SK       []ConfigSK  `yaml:"sk"`
	App      []ConfigApp `yaml:"app"`
	Redis    struct {
		Address string `yaml:"address"`
		DB      string `yaml:"db"`
	} `yaml:"redis"`
	SentryDSN string `yaml:"sentry_dsn"`
}

func (config Config) Check() (err error) {
	if len(config.SK) == 0 {
		return xerr.New("config.yaml sk 不能为空")
	}
	for index, sk := range config.SK {
		if strings.TrimSpace(sk.Value) == "" {
			return xerr.New(fmt.Sprintf("config.yaml sk[%d].value 不能为空", index))
		}
		if len(sk.Value) < 32 {
			return xerr.New(fmt.Sprintf("config.yaml sk[%d].value 长度不能小于32", index))
		}
	}
	for index, app := range config.App {
		if app.Appid == "" {
			return xerr.New(fmt.Sprintf("config.yaml app[%d].appid 不能为空", index))
		}
		if app.Secret == "" {
			return xerr.New(fmt.Sprintf("config.yaml app[%d].Secret 不能为空", index))
		}
	}
	if config.Redis.Address == "" {
		return xerr.New("config.yaml redis.address required")
	}
	if config.Redis.DB == "" {
		return xerr.New("config.yaml redis.db required")
	}
	return
}

type ConfigSK struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
type ConfigApp struct {
	Appid  string `yaml:"appid"`
	Secret string `yaml:"secret"`
}
