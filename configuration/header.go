package configuration

type Configuration struct {
	ConfigApi struct {
		Url      []string `yaml:"url" env:"true"`
		HostName string   `yaml:"hostname,omitempty" env:"true"`
		UserName string   `yaml:"username" env:"true"`
		Password string   `yaml:"password" env:"true"`
	} `yaml:"config_api"`
	Rtdb struct {
		Input  string `yaml:"input_bus"`
		Output string `yaml:"output_bus"`
	} `yaml:"rtdb,omitempty"`
	GridLosses struct {
		LogLevel    string `yaml:"log" env:"true"`
		QueueLength int    `yaml:"queue"`
		ApiPrefix   string `yaml:"api_prefix"`
	} `yaml:"grid_losses"`
}
