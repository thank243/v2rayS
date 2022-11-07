package cmd

import (
	"github.com/go-acme/lego/v4/lego"
	"github.com/urfave/cli"
)

func CreateFlags(defaultPath string) []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:  "accept-tos, a",
			Usage: "By setting this flag to true you indicate that you accept the current Let's Encrypt terms of service.",
		},
		cli.StringSliceFlag{
			Name:  "domains, d",
			Usage: "Add a domain to the process. Can be specified multiple times.",
		},
		cli.StringFlag{
			Name:  "email, m",
			Usage: "Email used for registration and recovery contact.",
		},
		cli.StringFlag{
			Name:  "dns",
			Usage: "Solve a DNS challenge using the specified provider. Can be mixed with other types of challenges. Run 'lego dnshelp' for help on usage.",
		},
		cli.BoolFlag{
			Name:  "http",
			Usage: "Use the HTTP challenge to solve challenges. Can be mixed with other types of challenges.",
		},
		cli.StringFlag{
			Name:  "server, s",
			Usage: "CA hostname (and optionally :port). The server certificate must be trusted in order to avoid further modifications to the client.",
			Value: lego.LEDirectoryProduction,
		},
		cli.StringFlag{
			Name:  "key-type, k",
			Value: "ec256",
			Usage: "Key type to use for private keys. Supported: rsa2048, rsa4096, rsa8192, ec256, ec384.",
		},
		cli.StringFlag{
			Name:   "path",
			EnvVar: "LEGO_PATH",
			Usage:  "Directory to use for storing the data.",
			Value:  defaultPath,
		},
		cli.StringFlag{
			Name:  "http.port",
			Usage: "Set the port and interface to use for HTTP based challenges to listen on.Supported: interface:port or :port.",
			Value: ":80",
		},
		cli.StringFlag{
			Name:  "http.proxy-header",
			Usage: "Validate against this HTTP header when solving HTTP based challenges behind a reverse proxy.",
			Value: "Host",
		},
		cli.StringFlag{
			Name:  "tls.port",
			Usage: "Set the port and interface to use for TLS based challenges to listen on. Supported: interface:port or :port.",
			Value: ":443",
		},
		cli.IntFlag{
			Name:  "dns-timeout",
			Usage: "Set the DNS timeout value to a specific value in seconds. Used only when performing authoritative name servers queries.",
			Value: 10,
		},
		cli.IntFlag{
			Name:  "cert.timeout",
			Usage: "Set the certificate timeout value to a specific value in seconds. Only used when obtaining certificates.",
			Value: 30,
		},
	}
}
