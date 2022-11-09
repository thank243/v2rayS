package mylego

import (
	"crypto"
	"crypto/x509"
	"log"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
)

func (l *LegoCMD) Renew() (bool, error) {
	account, client := l.newAccountsStorage().setup()
	l.setupChallenges(client)

	if account.Registration == nil {
		log.Panicf("Account %s is not registered. Use 'run' to register a new account.\n", account.Email)
	}

	return l.renewForDomains(client, NewCertificatesStorage(l.path))
}

func (l *LegoCMD) renewForDomains(client *lego.Client, certsStorage *CertificatesStorage) (bool, error) {
	// load the cert resource from files.
	// We store the certificate, private key and metadata in different files
	// as web servers would not be able to work with a combined file.
	certificates, err := certsStorage.ReadCertificate(l.C.CertDomain, ".crt")
	if err != nil {
		log.Panicf("Error while loading the certificate for domain %s\n\t%v", l.C.CertDomain, err)
	}

	cert := certificates[0]

	if !l.needRenewal(cert, 30) {
		return false, nil
	}

	// This is just meant to be informal for the user.
	timeLeft := cert.NotAfter.Sub(time.Now().UTC())
	newError("[%s] acme: Trying renewal with %d hours remaining", l.C.CertDomain, int(timeLeft.Hours())).AtWarning().WriteToLog()

	certDomains := certcrypto.ExtractDomains(cert)

	var privateKey crypto.PrivateKey
	request := certificate.ObtainRequest{
		Domains:    certDomains,
		PrivateKey: privateKey,
	}
	certRes, err := client.Certificate.Obtain(request)
	if err != nil {
		log.Panic(err)
	}

	certsStorage.SaveResource(certRes)

	return true, nil
}

func (l *LegoCMD) needRenewal(x509Cert *x509.Certificate, days int) bool {
	if x509Cert.IsCA {
		log.Panicf("[%s] Certificate bundle starts with a CA certificate", l.C.CertDomain)
	}

	if days >= 0 {
		notAfter := int(time.Until(x509Cert.NotAfter).Hours() / 24.0)
		if notAfter > days {
			log.Printf("[%s] The certificate expires in %d days, the number of days defined to perform the renewal is %d: no renewal.",
				l.C.CertDomain, notAfter, days)
			return false
		}
	}

	return true
}
