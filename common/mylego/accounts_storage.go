package mylego

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"golang.org/x/crypto/acme"
)

const (
	baseAccountsRootFolderName = "accounts"
	baseKeysFolderName         = "keys"
	accountFileName            = "account.json"
)

// AccountsStorage A storage for account data.
//
// rootPath:
//
//	./.lego/accounts/
//	     │      └── root accounts directory
//	     └── "path" option
//
// rootUserPath:
//
//	./.lego/accounts/localhost_14000/hubert@hubert.com/
//	     │      │             │             └── userID ("email" option)
//	     │      │             └── CA server ("server" option)
//	     │      └── root accounts directory
//	     └── "path" option
//
// keysPath:
//
//	./.lego/accounts/localhost_14000/hubert@hubert.com/keys/
//	     │      │             │             │           └── root keys directory
//	     │      │             │             └── userID ("email" option)
//	     │      │             └── CA server ("server" option)
//	     │      └── root accounts directory
//	     └── "path" option
//
// accountFilePath:
//
//	./.lego/accounts/localhost_14000/hubert@hubert.com/account.json
//	     │      │             │             │             └── account file
//	     │      │             │             └── userID ("email" option)
//	     │      │             └── CA server ("server" option)
//	     │      └── root accounts directory
//	     └── "path" option
type AccountsStorage struct {
	userID          string
	rootPath        string
	rootUserPath    string
	keysPath        string
	accountFilePath string
}

func (s *AccountsStorage) setup() (*Account, *lego.Client) {
	privateKey := s.GetPrivateKey(certcrypto.EC256)

	var account *Account
	if s.ExistsAccountFilePath() {
		account = s.LoadAccount(privateKey)
	} else {
		account = &Account{Email: s.GetUserID(), key: privateKey}
	}

	client := newClient(account, certcrypto.EC256)

	return account, client
}

func (s *AccountsStorage) ExistsAccountFilePath() bool {
	accountFile := filepath.Join(s.rootUserPath, accountFileName)
	if _, err := os.Stat(accountFile); os.IsNotExist(err) {
		return false
	} else if err != nil {
		log.Panic(err)
	}
	return true
}

func (s *AccountsStorage) GetRootPath() string {
	return s.rootPath
}

func (s *AccountsStorage) GetRootUserPath() string {
	return s.rootUserPath
}

func (s *AccountsStorage) GetUserID() string {
	return s.userID
}

func (s *AccountsStorage) Save(account *Account) error {
	jsonBytes, err := json.MarshalIndent(account, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(s.accountFilePath, jsonBytes, filePerm)
}

func (s *AccountsStorage) LoadAccount(privateKey crypto.PrivateKey) *Account {
	fileBytes, err := os.ReadFile(s.accountFilePath)
	if err != nil {
		log.Panicf("Could not load file for account %s: %v", s.userID, err)
	}

	var account Account
	err = json.Unmarshal(fileBytes, &account)
	if err != nil {
		log.Panicf("Could not parse file for account %s: %v", s.userID, err)
	}

	account.key = privateKey

	if account.Registration == nil || account.Registration.Body.Status == "" {
		reg, err := tryRecoverRegistration(privateKey)
		if err != nil {
			log.Panicf("Could not load account for %s. Registration is nil: %#v", s.userID, err)
		}

		account.Registration = reg
		err = s.Save(&account)
		if err != nil {
			log.Panicf("Could not save account for %s. Registration is nil: %#v", s.userID, err)
		}
	}

	return &account
}

func (s *AccountsStorage) GetPrivateKey(keyType certcrypto.KeyType) crypto.PrivateKey {
	accKeyPath := filepath.Join(s.keysPath, s.userID+".key")

	if _, err := os.Stat(accKeyPath); os.IsNotExist(err) {
		newError("No key found for account %s. Generating a %s key.", s.userID, keyType).WriteToLog()
		s.createKeysFolder()

		privateKey, err := generatePrivateKey(accKeyPath, keyType)
		if err != nil {
			log.Panicf("Could not generate RSA private account key for account %s: %v", s.userID, err)
		}

		newError("Saved key to %s", accKeyPath).WriteToLog()
		return privateKey
	}

	privateKey, err := loadPrivateKey(accKeyPath)
	if err != nil {
		log.Panicf("Could not load RSA private key from file %s: %v", accKeyPath, err)
	}

	return privateKey
}

func (s *AccountsStorage) createKeysFolder() {
	if err := createNonExistingFolder(s.keysPath); err != nil {
		log.Panicf("Could not check/create directory for account %s: %v", s.userID, err)
	}
}

func generatePrivateKey(file string, keyType certcrypto.KeyType) (crypto.PrivateKey, error) {
	privateKey, err := certcrypto.GeneratePrivateKey(keyType)
	if err != nil {
		return nil, err
	}

	certOut, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	defer certOut.Close()

	pemKey := certcrypto.PEMBlock(privateKey)
	err = pem.Encode(certOut, pemKey)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func loadPrivateKey(file string) (crypto.PrivateKey, error) {
	keyBytes, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	keyBlock, _ := pem.Decode(keyBytes)

	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(keyBlock.Bytes)
	}

	return nil, newError("unknown private key type").AtError()
}

func tryRecoverRegistration(privateKey crypto.PrivateKey) (*registration.Resource, error) {
	// couldn't load account but got a key. Try to look the account up.
	config := lego.NewConfig(&Account{key: privateKey})
	config.CADirURL = acme.LetsEncryptURL
	config.UserAgent = "lego-cli/dev"

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	reg, err := client.Registration.ResolveAccountByKey()
	if err != nil {
		return nil, err
	}
	return reg, nil
}
