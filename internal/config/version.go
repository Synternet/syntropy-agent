package config

var (
	version    = "0.0.0"
	subversion = "local"
)

func GetVersion() string {
	return GetFullVersion()
}

func GetFullVersion() string {
	if subversion != "" {
		return version + "-" + subversion
	}
	return version
}
