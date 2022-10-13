package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Config struct {
	Env             string `json:"env"`
	Port            int    `json:"port"`
	Host            string `json:"host"`
	RecordLimit     int    `json:"record_limit"`
	AttemptsAllowed int    `json:"attempts_allowed"`
}

func (c *Config) GetServerString() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func LoadConfig(path string) (Config, error) {

	configFile, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("can't open file: %v", err)
	}
	defer configFile.Close()

	configFileData, err := ioutil.ReadAll(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("can't convert to byte: %v", err)
	}

	var config Config
	if len(configFileData) != 0 {
		err = json.Unmarshal(configFileData, &config)
		if err != nil {
			return Config{}, fmt.Errorf("can't unmarshal json: %v", err)
		}
	}

	return config, nil
}
