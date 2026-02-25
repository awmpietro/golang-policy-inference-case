package config

import (
	"os"
	"strconv"
)

type Runtime struct {
	HTTPAddr       string
	CacheMaxItems  int
	PolicyMaxSteps int
	ObsBuffer      int
}

func Load() Runtime {
	return Runtime{
		HTTPAddr:       getenv("HTTP_ADDR", ":8080"),
		CacheMaxItems:  getenvInt("POLICY_CACHE_MAX_ITEMS", 1024, 1),
		PolicyMaxSteps: getenvInt("POLICY_MAX_STEPS", 10_000, 1),
		ObsBuffer:      getenvInt("POLICY_OBS_BUFFER", 4096, 1),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback, min int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < min {
		return fallback
	}
	return v
}
