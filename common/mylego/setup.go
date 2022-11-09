package mylego

import (
	"log"
	"os"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
	"golang.org/x/crypto/acme"
)

const filePerm os.FileMode = 0o600

func newClient(acc registration.User, keyType certcrypto.KeyType) *lego.Client {
	config := lego.NewConfig(acc)
	config.CADirURL = acme.LetsEncryptURL

	config.Certificate = lego.CertificateConfig{
		KeyType: keyType,
		Timeout: 30 * time.Second,
	}
	config.UserAgent = "lego-cli/dev"

	client, err := lego.NewClient(config)
	if err != nil {
		log.Panicf("Could not create client: %v", err)
	}

	return client
}

func (l *LegoCMD) setupChallenges(client *lego.Client) {
	switch l.C.CertMode {
	case "http":
		err := client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", ""))
		if err != nil {
			log.Panic(err)
		}
	case "tls":
		err := client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer("", ""))
		if err != nil {
			log.Panic(err)
		}
	case "dns":
		l.setupDNS(client)
	default:
		log.Panic("No challenge selected. You must specify at least one challenge: `http`, `tls`, `dns`.")
	}
}

func (l *LegoCMD) setupDNS(client *lego.Client) {
	provider, err := dns.NewDNSChallengeProviderByName(l.C.Provider)
	if err != nil {
		log.Panic(err)
	}

	err = client.Challenge.SetDNS01Provider(
		provider,
		dns01.CondOption(true, dns01.AddDNSTimeout(10*time.Second)),
	)
	if err != nil {
		log.Panic(err)
	}
}

func createNonExistingFolder(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0o700)
	} else if err != nil {
		return err
	}
	return nil
}
