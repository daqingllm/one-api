package env

import (
	"github.com/joho/godotenv"
	"github.com/songquanpeng/one-api/common"
	"os"
	"strconv"
)

func init() {
	godotenv.Load(*common.EnvFile)
}

func Bool(env string, defaultValue bool) bool {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	return os.Getenv(env) == "true"
}

func Int(env string, defaultValue int) int {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	num, err := strconv.Atoi(os.Getenv(env))
	if err != nil {
		return defaultValue
	}
	return num
}

func Float64(env string, defaultValue float64) float64 {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	num, err := strconv.ParseFloat(os.Getenv(env), 64)
	if err != nil {
		return defaultValue
	}
	return num
}

func String(env string, defaultValue string) string {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	return os.Getenv(env)
}
