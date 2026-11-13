package main

import (
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
		wantErr bool
	}{
		{
			name:    "splits shell arguments",
			command: `npm run dev -- --name "my service"`,
			want:    []string{"npm", "run", "dev", "--", "--name", "my service"},
		},
		{
			name:    "rejects empty command",
			command: "",
			wantErr: true,
		},
		{
			name:    "rejects whitespace command",
			command: " \t ",
			wantErr: true,
		},
		{
			name:    "rejects malformed quoting",
			command: `npm run "dev`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseCommand(%q) error = %v, wantErr %t", tt.command, err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCommand(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}
