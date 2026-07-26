package models

type RateLimitT struct {
	Route  string `yaml:"route"`
	Method string `yaml:"method"`
	Rate   int    `yaml:"rate"`
	Per    int    `yaml:"per"`
}

type Config struct {
	General struct {
		Host           string   `yaml:"host"`
		Port           int      `yaml:"port"`
		UseGzip        bool     `yaml:"use_gzip"`
		AllowedOrigins []string `yaml:"allowed_origins"`
	} `yaml:"general"`

	Cache struct {
		DSN string `yaml:"dsn"`
	} `yaml:"cache"`

	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`

	Pastes struct {
		MaxFiles    int `yaml:"max_files"`
		MaxFileSize int `yaml:"max_file_size"`
		IdLen       int `yaml:"id_length"`
		TokenLen    int `yaml:"token_length"`
	} `yaml:"pastes"`

	MessageQueue struct {
		DSN  string `yaml:"dsn"`
		Name string `yaml:"name"`
	} `yaml:"message_queue"`

	Limits []RateLimitT `yaml:"ratelimits"`
}
