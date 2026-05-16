package models

type Config struct {
	General struct {
		Host    string `yaml:"host"`
		Port    int    `yaml:"port"`
		UseGzip bool   `yaml:"use_gzip"`
	} `yaml:"general"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
}
