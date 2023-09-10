package core

import (
	"fmt"
	"github.com/gin-contrib/cors"
)

type Config struct {
	Port        int
	Host        string
	Debug       bool
	CorsOptions cors.Config
	JwtSecret   []byte
}

func (c *Config) GetServerString() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
