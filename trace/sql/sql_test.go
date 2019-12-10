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
	}

	for _, tc := range testcases {
		got, _ := ParseDSN(tc.dsn)
		assert.Equal(t, got, tc.want)
	}
}
