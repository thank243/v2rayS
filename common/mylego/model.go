package mylego

type CertConfig struct {
	CertMode                string            `mapstructure:"CertMode"` // none, file, http, dns
	VerifyClientCertificate bool              `mapstructure:"VerifyClientCertificate"`
	CertDomain              string            `mapstructure:"CertDomain"`
	CertFile                string            `mapstructure:"CertFile"`
	KeyFile                 string            `mapstructure:"KeyFile"`
	Provider                string            `mapstructure:"Provider"` // alidns, cloudflare, gandi, godaddy....
	Email                   string            `mapstructure:"Email"`
	DNSEnv                  map[string]string `mapstructure:"DNSEnv"`
}

type LegoCMD struct {
	C    *CertConfig
	path string
}
