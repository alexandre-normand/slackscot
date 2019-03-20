package plugins_test

import (
	"github.com/stretchr/testify/mock"
)

// mockStorer holds a mock to implement of mock of StringStorer
type mockStorer struct {
	mock.Mock
}

// GetString mocks an implementation of GetString
func (ms *mockStorer) GetString(key string) (value string, err error) {
	args := ms.Called(key)

	return args.String(0), args.Error(1)
}

// GetSiloString mocks an implementation of GetSiloString
func (ms *mockStorer) GetSiloString(silo string, key string) (value string, err error) {
	args := ms.Called(silo, key)

	return args.String(0), args.Error(1)
}

// PutString mocks an implementation of PutString
func (ms *mockStorer) PutString(key string, value string) (err error) {
	args := ms.Called(key, value)

	return args.Error(0)
}

// PutSiloString mocks an implementation of PutSiloString
func (ms *mockStorer) PutSiloString(silo string, key string, value string) (err error) {
	args := ms.Called(silo, key, value)

	return args.Error(0)
}

// DeleteString mocks an implementation of DeleteString
func (ms *mockStorer) DeleteString(key string) (err error) {
	args := ms.Called(key)

	return args.Error(0)
}

// DeleteSiloString mocks an implementation of DeleteSiloString
func (ms *mockStorer) DeleteSiloString(silo string, key string) (err error) {
	args := ms.Called(silo, key)

	return args.Error(0)
}

// Scan mocks an implementation of Scan
func (ms *mockStorer) Scan() (entries map[string]string, err error) {
	args := ms.Called()

	return args.Get(0).(map[string]string), args.Error(1)
}

// ScanSilo mocks an implementation of ScanSilo
func (ms *mockStorer) ScanSilo(silo string) (entries map[string]string, err error) {
	args := ms.Called(silo)

	return args.Get(0).(map[string]string), args.Error(1)
}

// Close mocks an implementation of Close
func (ms *mockStorer) Close() (err error) {
	args := ms.Called()

	return args.Error(0)
}
