package core

import (
	"fmt"
	"github.com/rs/cors"
)

type Config struct {
	Port        int
	Host        string
	Debug       bool
	CorsOptions cors.Options
}

func (c *Config) GetServerString() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
