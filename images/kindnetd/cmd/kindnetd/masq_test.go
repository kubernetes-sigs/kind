package main

import (
	"reflect"
	"testing"
)

func TestValidateNoMasqueradeCIDRs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ipv6    bool
		cidrs   []string
		want    []string
		wantErr bool
	}{
		{
			name:  "valid ipv4 cidrs",
			cidrs: []string{"10.244.0.0/16", "10.96.0.0/12"},
			want:  []string{"10.244.0.0/16", "10.96.0.0/12"},
		},
		{
			name:  "trim whitespace",
			cidrs: []string{" 10.244.0.0/16 ", "\t10.96.0.0/12\n"},
			want:  []string{"10.244.0.0/16", "10.96.0.0/12"},
		},
		{
			name:  "valid ipv6 cidrs",
			ipv6:  true,
			cidrs: []string{"fd00:10:244::/56", "fd00:10:96::/112"},
			want:  []string{"fd00:10:244::/56", "fd00:10:96::/112"},
		},
		{
			name:    "invalid cidr",
			cidrs:   []string{"not-a-cidr"},
			wantErr: true,
		},
		{
			name:    "ipv4 agent rejects ipv6 cidr",
			cidrs:   []string{"fd00:10:244::/56"},
			wantErr: true,
		},
		{
			name:    "ipv6 agent rejects ipv4 cidr",
			ipv6:    true,
			cidrs:   []string{"10.244.0.0/16"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := validateNoMasqueradeCIDRs(tt.ipv6, tt.cidrs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("validateNoMasqueradeCIDRs() = %v, want %v", got, tt.want)
			}
		})
	}
}
