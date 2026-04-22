package config

import (
	"time"
)

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	ExposeHeaders    []string
	MaxAge           time.Duration
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3000",      // local dev (React)
			"http://localhost:5173",      // Vite local
			"https://your-frontend.com",  // production domain
			"https://www.your-frontend.com",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"PATCH",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"Accept",
		},
		AllowCredentials: true, // ВАЖНО: если используете cookies (refresh_token)
		ExposeHeaders: []string{
			"Content-Length",
			"Authorization",
		},
		MaxAge: 12 * time.Hour, // Prepost-запросы кэшируются 12 часов
	}
}