package config

import "github.com/LlirikP/pr_dispenser/internal/database"

type ApiConfig struct {
	DB *database.Queries
}

var ApiCfg *ApiConfig
