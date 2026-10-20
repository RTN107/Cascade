// Package config reads all runtime settings from environment variables.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
	WebOrigin   string
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Port:        os.Getenv("API_PORT"),
		WebOrigin:   os.Getenv("WEB_ORIGIN"),
	}
	if c.Port == "" {
		// most hosting platforms, Render included, assign the port to listen on
		// through PORT rather than this project's own API_PORT
		c.Port = os.Getenv("PORT")
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.WebOrigin == "" {
		c.WebOrigin = "http://localhost:5173"
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is not set")
	}
	if c.JWTSecret == "" {
		return c, fmt.Errorf("JWT_SECRET is not set")
	}
	return c, nil
}
