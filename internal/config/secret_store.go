package config

import (
	"errors"
	"strings"

	"github.com/zalando/go-keyring"
)

const defaultSecretStoreService = "commit-now-myfriend"

type SystemSecretStore struct {
	Service string
}

func NewSystemSecretStore() SystemSecretStore {
	return SystemSecretStore{Service: defaultSecretStoreService}
}

func (s SystemSecretStore) GetAPIKey(provider ProviderType) (*string, error) {
	value, err := go_keyringGet(s.service(), secretStoreAccount(provider))
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		// Reading credentials is optional during config resolution. A missing or
		// unavailable platform store should behave like an absent key so commands
		// can still report configuration state and fall back to env/plaintext config.
		return nil, nil
	}
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return &value, nil
}

func (s SystemSecretStore) SetAPIKey(provider ProviderType, apiKey string) error {
	if strings.TrimSpace(apiKey) == "" {
		return newError("Cannot store an empty API key for %s.", provider)
	}
	return go_keyringSet(s.service(), secretStoreAccount(provider), apiKey)
}

func (s SystemSecretStore) service() string {
	if strings.TrimSpace(s.Service) != "" {
		return s.Service
	}
	return defaultSecretStoreService
}

func secretStoreAccount(provider ProviderType) string {
	return "cnm:" + string(provider)
}

var go_keyringGet = keyring.Get
var go_keyringSet = keyring.Set
