package sql

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseDSN(t *testing.T) {
	type testcase struct {
		dsn     string
		want    DSNData
		wantErr error
	}
	var testcases = []testcase{
		{"username:password@protocol(address)/dbname?param=value", DSNData{"dbname", "address", "username", "protocol", "password"}, nil},
		{"bruce:hunter2@tcp(127.0.0.1)/arkhamdb?param=value", DSNData{"arkhamdb", "127.0.0.1", "bruce", "tcp", "hunter2"}, nil},
	}

	for _, tc := range testcases {
		got, _ := ParseDSN(tc.dsn)
		assert.Equal(t, got, tc.want)
	}
}

func BenchmarkParseDSN(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ParseDSN("bruce:hunter2@tcp(127.0.0.1)/arkhamdb?param=value")
	}
}
