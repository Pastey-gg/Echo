package models

type Config struct {
	General struct {
		Host           string   `yaml:"host"`
		Port           int      `yaml:"port"`
		UseGzip        bool     `yaml:"use_gzip"`
		AllowedOrigins []string `yaml:"allowed_origins"`
	} `yaml:"general"`

	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`

	Pastes struct {
		MaxFiles    int `yaml:"max_files"`
		MaxFileSize int `yaml:"max_file_size"`
		IdLen       int `yaml:"id_length"`
		TokenLen    int `yaml:"token_length"`
	} `yaml:"pastes"`
}
