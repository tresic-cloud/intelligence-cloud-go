package profile

import "github.com/zalando/go-keyring"

// realKeyring wraps the zalando/go-keyring package to implement KeyringBackend.
type realKeyring struct{}

func (r *realKeyring) Get(service, account string) (string, error) {
	pw, err := keyring.Get(service, account)
	if err != nil {
		return "", mapKeyringError(err)
	}
	return pw, nil
}

func (r *realKeyring) Set(service, account, password string) error {
	if err := keyring.Set(service, account, password); err != nil {
		return mapKeyringError(err)
	}
	return nil
}

func (r *realKeyring) Delete(service, account string) error {
	if err := keyring.Delete(service, account); err != nil {
		return mapKeyringError(err)
	}
	return nil
}

// mapKeyringError translates go-keyring errors to our sentinel errors.
func mapKeyringError(err error) error {
	if err == nil {
		return nil
	}
	switch err {
	case keyring.ErrNotFound:
		return ErrKeyringNotFound
	case keyring.ErrUnsupportedPlatform:
		return ErrKeyringUnsupported
	default:
		return err
	}
}
