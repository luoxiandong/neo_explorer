package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"net"
	"net/url"
	"strings"
)

type config struct {
	// MySQL configs.
	User     string
	Password string
	Hostname string
	Port     string
	Database string

	// Label sets log output prefix.
	Label string

	RPCs []string `mapstructure:"rpc_url"`

	// Workers sets the number of goroutines that will be created for data processing.
	// Recommend value: 3.
	Workers int
}

var cfg config

func Load() {
	viper.SetConfigName("config") // 设置配置文件名
	viper.AddConfigPath(".")      // 第一个搜索路径
	err := viper.ReadInConfig()   // 读取配置数据
	if err != nil {
		panic(err)
	}

	err = viper.Unmarshal(&cfg) // 将配置信息绑定到结构体上
	if err != nil {
		panic(err)
	}

	err = check() // check config
	if err != nil {
		panic(err)
	}
}

func check() error {
	if cfg.Workers < 1 {
		return errors.New("value of 'goroutine' must greater than or equal to 1")
	}

	if len(cfg.RPCs) < 1 {
		return errors.New("at least 1 rpc server url must be set")
	}

	for _, rpc := range cfg.RPCs {
		if strings.HasPrefix(rpc, "http") {
			u, err := url.Parse(rpc)
			if err != nil {
				return err
			}
			rpc = u.Host
		}

		_, _, err := net.SplitHostPort(rpc)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetDbConnStr returns mysql connection string.
func GetDbConnStr() string {
	str := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s",
		cfg.User,
		cfg.Password,
		cfg.Hostname,
		cfg.Port,
		cfg.Database,
	)

	params := []string{
		"charset=utf8",
		"parseTime=True",
		"loc=Local",
		"maxAllowedPacket=52428800",
		"multiStatements=True",
	}

	if len(params) > 0 {
		str = fmt.Sprintf("%s?%s", str, strings.Join(params, "&"))
	}

	return str
}

// GetLabel returns custome label as console output prefix.
func GetLabel() string {
	return cfg.Label
}

// GetRPCs returns all rpc urls from config.
func GetRPCs() []string {
	return cfg.RPCs
}

// GetGoroutines returns the number of working goroutines.
func GetGoroutines() int {
	return cfg.Workers
}
