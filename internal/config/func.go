package config

import (
	"os"
	"strconv"
)

func initUint(variable *uint, name string, defaultValue uint) {
	str := os.Getenv(name)
	val, err := strconv.Atoi(str)
	if len(str) == 0 || err != nil {
		*variable = defaultValue
		return
	}
	*variable = uint(val)
}

func initBool(variable *bool, name string, defaultValue bool) {
	str := os.Getenv(name)
	val, err := strconv.ParseBool(str)
	if len(str) == 0 || err != nil {
		*variable = defaultValue
		return
	}
	*variable = val
}

func initString(variable *string, name string, defaultValue string) {
	str := os.Getenv(name)
	if len(str) == 0 {
		*variable = defaultValue
		return
	}
	*variable = str
}
