// Package mocks contains a mock of the store package interfaces
package mocks

import (
	"github.com/stretchr/testify/mock"
)

// Storer holds a mock to implement of mock of StringStorer
type Storer struct {
	mock.Mock
}

// GetString mocks an implementation of GetString
func (ms *Storer) GetString(key string) (value string, err error) {
	args := ms.Called(key)

	return args.String(0), args.Error(1)
}

// GetSiloString mocks an implementation of GetSiloString
func (ms *Storer) GetSiloString(silo string, key string) (value string, err error) {
	args := ms.Called(silo, key)

	return args.String(0), args.Error(1)
}

// PutString mocks an implementation of PutString
func (ms *Storer) PutString(key string, value string) (err error) {
	args := ms.Called(key, value)

	return args.Error(0)
}

// PutSiloString mocks an implementation of PutSiloString
func (ms *Storer) PutSiloString(silo string, key string, value string) (err error) {
	args := ms.Called(silo, key, value)

	return args.Error(0)
}

// DeleteString mocks an implementation of DeleteString
func (ms *Storer) DeleteString(key string) (err error) {
	args := ms.Called(key)

	return args.Error(0)
}

// DeleteSiloString mocks an implementation of DeleteSiloString
func (ms *Storer) DeleteSiloString(silo string, key string) (err error) {
	args := ms.Called(silo, key)

	return args.Error(0)
}

// Scan mocks an implementation of Scan
func (ms *Storer) Scan() (entries map[string]string, err error) {
	args := ms.Called()

	return args.Get(0).(map[string]string), args.Error(1)
}

// ScanSilo mocks an implementation of ScanSilo
func (ms *Storer) ScanSilo(silo string) (entries map[string]string, err error) {
	args := ms.Called(silo)

	return args.Get(0).(map[string]string), args.Error(1)
}

// GlobalScan mocks an implementation of GlobalScan
func (ms *Storer) GlobalScan() (entries map[string]map[string]string, err error) {
	args := ms.Called()

	return args.Get(0).(map[string]map[string]string), args.Error(1)
}

// Close mocks an implementation of Close
func (ms *Storer) Close() (err error) {
	args := ms.Called()

	return args.Error(0)
}
