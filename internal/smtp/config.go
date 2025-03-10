package smtp

type Config struct {
	User             string
	Password         string
	Host             string
	Port             int
	From             string
	AllowInsecureTls bool
}
