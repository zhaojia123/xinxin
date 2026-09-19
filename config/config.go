package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"friends-records/internal/apperror"
	"github.com/BurntSushi/toml"
)

type Config struct {
	App    AppConfig    `toml:"app"`
	MySQL  MySQLConfig  `toml:"mysql"`
	WeChat WeChatConfig `toml:"wechat"`
	Upload UploadConfig `toml:"upload"`
}
type AppConfig struct {
	Env             string `toml:"env"`
	Addr            string `toml:"addr"`
	TokenSecret     string `toml:"token_secret"`
	FilingNumber    string `toml:"filing_number"`
	FilingLanding   bool   `toml:"filing_landing"`
	MonthlyRestDays int    `toml:"monthly_rest_days"`
}
type MySQLConfig struct {
	DSN string `toml:"dsn"`
}
type WeChatConfig struct {
	AppID        string `toml:"app_id"`
	AppSecret    string `toml:"app_secret"`
	MockLogin    bool   `toml:"mock_login"`
	AutoRegister bool   `toml:"auto_register"`
}
type UploadConfig struct {
	Dir       string `toml:"dir"`
	MaxSizeMB int64  `toml:"max_size_mb"`
}

func Load(name string) (Config, error) {
	path, err := find(name)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	metadata, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return Config{}, apperror.Wrap(err, fmt.Sprintf("读取配置文件%s失败", path))
	}
	if rest := metadata.Undecoded(); len(rest) > 0 {
		return Config{}, apperror.New(fmt.Sprintf("配置项%s无法识别，请检查拼写", rest[0].String()))
	}
	if cfg.App.Env == "" {
		cfg.App.Env = "development"
	}
	if cfg.App.Addr == "" {
		cfg.App.Addr = ":8080"
	}
	if cfg.App.MonthlyRestDays <= 0 {
		cfg.App.MonthlyRestDays = 3
	}
	if cfg.App.MonthlyRestDays >= 31 {
		return Config{}, apperror.New("配置app.monthly_rest_days必须小于31")
	}
	// 临时联调或同机多实例运行时，可用环境变量覆盖监听端口。
	if addr := strings.TrimSpace(os.Getenv("FRIENDS_RECORDS_ADDR")); addr != "" {
		cfg.App.Addr = addr
	}
	if cfg.Upload.Dir == "" {
		cfg.Upload.Dir = "data/uploads"
	}
	if cfg.Upload.MaxSizeMB <= 0 {
		cfg.Upload.MaxSizeMB = 5
	}
	if cfg.Upload.MaxSizeMB > 20 {
		return Config{}, apperror.New("配置upload.max_size_mb不能大于20")
	}
	if cfg.App.TokenSecret != "" && len(cfg.App.TokenSecret) < 32 {
		return Config{}, apperror.New("配置app.token_secret留空或填写至少32个字符")
	}
	if strings.EqualFold(cfg.App.Env, "production") && len(cfg.App.TokenSecret) < 32 {
		return Config{}, apperror.New("生产环境必须填写不少于32个字符的app.token_secret")
	}
	if cfg.WeChat.AppID == "" {
		return Config{}, apperror.New("配置wechat.app_id不能为空")
	}
	if strings.EqualFold(cfg.App.Env, "production") && cfg.WeChat.MockLogin {
		return Config{}, apperror.New("生产环境必须关闭wechat.mock_login")
	}
	if !cfg.WeChat.MockLogin && cfg.WeChat.AppSecret == "" {
		return Config{}, apperror.New("关闭模拟登录后wechat.app_secret不能为空")
	}
	return cfg, nil
}
func find(name string) (string, error) {
	if filepath.IsAbs(name) {
		return name, nil
	}
	directory, err := os.Getwd()
	if err != nil {
		return "", apperror.Wrap(err, "获取程序当前目录失败")
	}
	for {
		candidate := filepath.Join(directory, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", apperror.New(fmt.Sprintf("找不到配置文件%s，请在项目目录或子目录中启动程序", name))
}
