package config

// AppConfig menampung konfigurasi aplikasi dari .env.
// Sengaja sederhana: hanya yang dibutuhkan tugas ini.
type AppConfig struct {
	Port string
}

func LoadAppConfig() AppConfig {
	LoadEnv()
	return AppConfig{
		Port: GetEnv("APP_PORT", "3000"),
	}
}
