package patron

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	type args struct {
		cmp []Component
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"success", args{cmp: []Component{&testComponent{}}}, false},
		{"failed missing components", args{cmp: nil}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := new(tt.args.cmp)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestServer_Run_Shutdown(t *testing.T) {
	tests := []struct {
		name    string
		cp      Component
		wantErr bool
	}{
		{name: "success", cp: &testComponent{}, wantErr: false},
		{name: "failed to run", cp: &testComponent{errorRunning: true}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := new([]Component{tt.cp, tt.cp, tt.cp})
			assert.NoError(t, err)
			err = s.Run()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type testComponent struct {
	errorRunning bool
}

func (ts testComponent) Run(ctx context.Context) error {
	if ts.errorRunning {
		return errors.New("failed to run component")
	}
	return nil
}

func (ts testComponent) Info() map[string]interface{} {
	return map[string]interface{}{"type": "mock"}
}
