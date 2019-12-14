package sql

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseDSN(t *testing.T) {
	type testcase struct {
		dsn  string
		want DSNInfo
	}

	//TODO: restructure table-driven tests to the proposed way
	var testcases = []testcase{
		{"username:password@protocol(address)/dbname?param=value", DSNInfo{"", "dbname", "address", "username", "protocol", "password"}},
		{"/", DSNInfo{"", "", "", "", "", ""}},
		{"/dbname", DSNInfo{"", "dbname", "", "", "", ""}},
		{"user:p@/ssword@/", DSNInfo{"", "", "", "user", "", "p@/ssword"}},
		{"mysql://user:p@/ssword@/", DSNInfo{"mysql://", "", "", "user", "", "p@/ssword"}},
		{"postgresql://user:p@/ssword@/", DSNInfo{"postgresql://", "", "", "user", "", "p@/ssword"}},
		{"user@unix(/path/to/socket)/dbname?charset=utf8", DSNInfo{"", "dbname", "/path/to/socket", "user", "unix", ""}},
		{"user:password@/dbname?param1=val1&param2=val2&param3=val3", DSNInfo{"", "dbname", "", "user", "", "password"}},
		{"bruce:hunter2@tcp(127.0.0.1)/arkhamdb?param=value", DSNInfo{"", "arkhamdb", "127.0.0.1", "bruce", "tcp", "hunter2"}},
		{"user@unix(/path/to/mydir@/socket)/dbname?charset=utf8", DSNInfo{"", "dbname", "/path/to/mydir@/socket", "user", "unix", ""}},
		{"user:password@tcp(localhost:5555)/dbname?charset=utf8&tls=true", DSNInfo{"", "dbname", "localhost:5555", "user", "tcp", "password"}},
		{"us:er:name:password@memory(localhost:5555)/dbname?charset=utf8&tls=true", DSNInfo{"", "dbname", "localhost:5555", "us", "memory", "er:name:password"}},
		{"user:p@ss(word)@tcp([c023:9350:225b:671a:2cdd:3d83:7c19:ca42]:80)/dbname?loc=Local", DSNInfo{"", "dbname", "[c023:9350:225b:671a:2cdd:3d83:7c19:ca42]:80", "user", "tcp", "p@ss(word)"}},
		{"", DSNInfo{"", "", "", "", "", ""}},
		{"rosebud", DSNInfo{"", "", "", "", "", ""}},
	}

	for _, tc := range testcases {
		got := ParseDSN(tc.dsn)
		assert.Equal(t, got, tc.want)
	}
}

func BenchmarkParseDSN(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ParseDSN("bruce:hunter2@tcp(127.0.0.1)/arkhamdb?param=value")
	}
}
