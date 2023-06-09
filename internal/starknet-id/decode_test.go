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
