package patron

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSIGHUP(t *testing.T) {
	type args struct {
		handler func()
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "nil handler", args: args{handler: nil}, wantErr: true},
		{name: "success", args: args{handler: func() {}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := new([]Component{&testComponent{}})
			assert.NoError(t, err)
			err = sighub(tt.args.handler)(s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
