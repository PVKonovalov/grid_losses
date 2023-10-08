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
		Losses      []struct {
			Equipment int    `yaml:"equipment"`
			VoltageAc uint64 `yaml:"voltage_ac"`
			CurrentA  uint64 `yaml:"current_a"`
			CosPhi    uint64 `yaml:"cos_phi"`
			State     uint64 `yaml:"state"`
		} `yaml:"losses"`
	} `yaml:"grid_losses"`
}
