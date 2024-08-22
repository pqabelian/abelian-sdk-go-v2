package abelian

const (
	defaultLogFilename  = "abelian-sdk-go.log"
	defaultConfFilename = "abelian-sdk-go-default.conf"
)

type Config struct {
	NetID string `json:"networkID"`

	LogDir   string `json:"logDir,omitempty"`
	LogLevel string `json:"logLevel,omitempty"`
}

//func applyConfig(cfg *Config) error {
//	chain.initNetWorkID(cfg.NetID)
//	logger.initLogRotator(filepath.Join(cfg.LogDir, defaultLogFilename))
//	logger.setLogLevels(cfg.LogLevel)
//	return nil
//}

var conf *Config

func GetConf() *Config {
	return conf
}

//func init() {
//	file, err := os.ReadFile(defaultConfFilename)
//	if err != nil {
//		panic(fmt.Errorf("fail to load config file: %v", err))
//	}
//	err = json.Unmarshal(file, &conf)
//	if err != nil {
//		panic(fmt.Errorf("fail to load config file: %v", err))
//	}
//}
