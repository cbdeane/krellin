package agents

import "github.com/zalando/go-keyring"

type SecretStore interface {
	Get(providerName string) (string, error)
	Set(providerName string, secret string) error
	Delete(providerName string) error
}

type KeyringStore struct {
	Service string
}

func NewKeyringStore() *KeyringStore {
	return &KeyringStore{Service: "krellin"}
}

func (k *KeyringStore) Get(providerName string) (string, error) {
	return keyring.Get(k.Service, providerKey(providerName))
}

func (k *KeyringStore) Set(providerName string, secret string) error {
	return keyring.Set(k.Service, providerKey(providerName), secret)
}

func (k *KeyringStore) Delete(providerName string) error {
	return keyring.Delete(k.Service, providerKey(providerName))
}

func providerKey(name string) string {
	return "provider:" + name
}
