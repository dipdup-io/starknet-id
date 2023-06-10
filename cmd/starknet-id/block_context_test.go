package main

import (
	"reflect"
	"testing"

	"github.com/dipdup-io/starknet-go-api/pkg/data"
)

func TestBlockContext_decodeDomainName(t *testing.T) {
	tests := []struct {
		name    string
		domains []data.Felt
		want    []string
		wantErr bool
	}{
		{
			name: "deployer.fricoben",
			domains: []data.Felt{
				data.Felt("0x1c81fe3d15f"),
				data.Felt("0x15d246f6c1b"),
			},
			want: []string{
				"deployer", "fricoben",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := new(BlockContext)
			got, err := bc.decodeDomainName(tt.domains)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlockContext.decodeDomainName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BlockContext.decodeDomainName() = %v, want %v", got, tt.want)
			}
		})
	}
}
