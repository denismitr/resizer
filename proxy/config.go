package proxy

import "time"

type Config struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}
