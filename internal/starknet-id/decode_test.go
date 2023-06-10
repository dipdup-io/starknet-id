package starknetid

import (
	"testing"

	"github.com/dipdup-io/starknet-go-api/pkg/data"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		f       data.Felt
		want    string
		wantErr bool
	}{
		{
			name: "cat",
			f:    data.Felt("0x6B2E"),
			want: "cat",
		}, {
			name: "cryptoalka1",
			f:    data.Felt("0x25A62B324F0CD00"),
			want: "cryptoalka1",
		}, {
			name: "coinify",
			f:    data.Felt("0x10EBD49A0E"),
			want: "coinify",
		}, {
			name: "xplorer",
			f:    data.Felt("0xbfff81efd"),
			want: "xplorer",
		}, {
			name: "adalia",
			f:    data.Felt("0xafc5f49a"),
			want: "adalia",
		}, {
			name: "fricoben",
			f:    data.Felt("0x15d246f6c1b"),
			want: "fricoben",
		}, {
			name: "deployer",
			f:    data.Felt("0x1c81fe3d15f"),
			want: "deployer",
		}, {
			name: "oj1fcb这r",
			f:    data.Felt("0x3a3b3079e28"),
			want: "oj1fcb这r",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Decode() = %v, want %v", got, tt.want)
			}
		})
	}
}
