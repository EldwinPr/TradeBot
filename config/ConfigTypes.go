package config

type config struct {
	Exchange ExchangeConfig
	Database DatabaseConfig
	Symbols  []string
}

type ExchangeConfig struct {
	APIKey    string
	SecretKey string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}
