package config

type config struct {
	url         string
	smtp        string
	fromAddress string
}

var conf = config{
	url:         "http://localhost:8080",
	smtp:        "localhost:25",
	fromAddress: "weters19@gmail.com",
}

func Load() error {
	return nil
}

func GetURL(optionalPath ...string) string {
	if len(optionalPath) > 0 {
		return conf.url + optionalPath[0]
	}

	return conf.url + "/"
}

func GetSMTP() string {
	return conf.smtp
}

func GetFromAddress() string {
	return conf.fromAddress
}
