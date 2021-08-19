package config

var (
	version    = "0.0.0"
	subversion = "local"
)

func GetVersion() string {
	return version
}

func GetFullVersion() string {
	if subversion != "" {
		return version + "-" + subversion
	}
	return version
}
