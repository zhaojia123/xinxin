package config

import (
	"os"
	"path/filepath"
	"testing"
)

const testConfig = `[app]
env = "development"
addr = ":8999"

[mysql]
dsn = "test"

[wechat]
app_id = "test-app"
mock_login = true
`

func TestLoadDefaultPrefersLocalConfig(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, filepath.Join(dir, "config.local.toml"))
	writeProductionConfig(t, filepath.Join(dir, "config.production.toml"))
	switchWorkingDirectory(t, dir)
	t.Setenv(configPathEnv, "")

	cfg, err := LoadDefault()
	if err != nil {
		t.Fatalf("加载默认配置失败：%v", err)
	}
	if cfg.App.Env != "development" {
		t.Fatalf("本地配置未被优先读取：%s", cfg.App.Env)
	}
}

func TestLoadDefaultUsesEnvironmentOverride(t *testing.T) {
	dir := t.TempDir()
	production := filepath.Join(dir, "production.toml")
	if err := os.WriteFile(production, []byte(`[app]
env = "production"
addr = ":8999"
token_secret = "12345678901234567890123456789012"

[mysql]
dsn = "test"

[wechat]
app_id = "test-app"
app_secret = "test-secret"
mock_login = false
`), 0o600); err != nil {
		t.Fatalf("写入测试配置失败：%v", err)
	}
	t.Setenv(configPathEnv, production)

	cfg, err := LoadDefault()
	if err != nil {
		t.Fatalf("按环境变量加载配置失败：%v", err)
	}
	if cfg.App.Env != "production" {
		t.Fatalf("环境变量指定的配置未生效：%s", cfg.App.Env)
	}
}

func writeConfig(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(testConfig), 0o600); err != nil {
		t.Fatalf("写入测试配置失败：%v", err)
	}
}

func writeProductionConfig(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(`[app]
env = "production"
addr = ":8999"
token_secret = "12345678901234567890123456789012"

[mysql]
dsn = "test"

[wechat]
app_id = "test-app"
app_secret = "test-secret"
mock_login = false
`), 0o600); err != nil {
		t.Fatalf("写入生产测试配置失败：%v", err)
	}
}

func switchWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("读取当前目录失败：%v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("切换测试目录失败：%v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("恢复测试目录失败：%v", err)
		}
	})
}
