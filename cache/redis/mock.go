package redis

import "time"

type kv struct {
	value string
	ttl   time.Duration
}

// MockClient mocks the Redis store.
type MockClient struct {
	data map[string]kv
}

// NewMockRedis initializes the MockClient.
func NewMockRedis() *MockClient {
	return &MockClient{make(map[string]kv)}
}

// Contains performs a lookup on the mock client data.
func (m *MockClient) Contains(key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

// Get performs a lookup on the mock client data and returns the value.
func (m *MockClient) Get(key string) (interface{}, bool, error) {
	d, ok := m.data[key]
	return d.value, ok, nil
}

// Purge clears out the mock client data.
func (m *MockClient) Purge() error {
	m.data = make(map[string]kv)
	return nil
}

// Remove deletes a key from the mock client data.
func (m *MockClient) Remove(key string) error {
	delete(m.data, key)
	return nil
}

// Set sets the value on a stored mock client entry.
func (m *MockClient) Set(key string, value interface{}) error {
	m.data[key] = kv{value.(string), 0}
	return nil
}

// SetTTL sets the value on a stored mock client entry, also setting a TTL parameter.
func (m *MockClient) SetTTL(key string, value interface{}, ttl time.Duration) error {
	m.data[key] = kv{value.(string), ttl}
	return nil
}
