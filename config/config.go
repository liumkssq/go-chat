package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 配置信息
type Config struct {
	App        *App        `json:"app" yaml:"app"`
	Redis      *Redis      `json:"redis" yaml:"redis"`
	MySQL      *MySQL      `json:"mysql" yaml:"mysql"`
	Jwt        *Jwt        `json:"jwt" yaml:"jwt"`
	Cors       *Cors       `json:"cors" yaml:"cors"`
	Log        *Log        `json:"log" yaml:"log"`
	Filesystem *Filesystem `json:"filesystem" yaml:"filesystem"`
	Email      *Email      `json:"email" yaml:"email"`
	Server     *Server     `json:"server" yaml:"server"`
	Nsq        *Nsq        `json:"nsq" yaml:"nsq"` // 目前没用到
}

type Server struct {
	Http      int `json:"http" yaml:"http"`
	Websocket int `json:"websocket" yaml:"websocket"`
	Tcp       int `json:"tcp" yaml:"tcp"`
}

func New(filename string) *Config {
	content, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var conf Config
	fmt.Printf("config.yaml: +%v\n", conf)
	if err := yaml.Unmarshal(content, &conf); err != nil {
		fmt.Printf("解析 config.yaml 读取错误: %v\n", err)
		fmt.Printf("config.yaml: +%v\n", conf)
		panic(fmt.Sprintf("解析 config.yaml 读取错误: %v", err))
	}
	fmt.Printf("config.yaml: +%v\n", conf)
	return &conf
}

// Debug 调试模式
func (c *Config) Debug() bool {
	return c.App.Debug
}
