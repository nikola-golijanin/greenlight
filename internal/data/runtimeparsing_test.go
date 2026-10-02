package data

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestRuntimeMarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		runtime Runtime
		want    string
	}{
		{"typical value", 107, `"107 mins"`},
		{"zero", 0, `"0 mins"`},
		{"one", 1, `"1 mins"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.runtime)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}


func TestRuntimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Runtime
		wantErr error
	}{
		{name: "valid", input: `"107 mins"`, want: 107},
		{name: "zero", input: `"0 mins"`, want: 0},
		{name: "unquoted number", input: `107`, wantErr: ErrInvalidRuntimeFormat},
		{name: "number as string without unit", input: `"107"`, wantErr: ErrInvalidRuntimeFormat},
		{name: "wrong unit", input: `"107 minutes"`, wantErr: ErrInvalidRuntimeFormat},
		{name: "non-numeric value", input: `"abc mins"`, wantErr: ErrInvalidRuntimeFormat},
		{name: "double space", input: `"107  mins"`, wantErr: ErrInvalidRuntimeFormat},
		{name: "empty string", input: `""`, wantErr: ErrInvalidRuntimeFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Runtime
			err := json.Unmarshal([]byte(tt.input), &got)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got error %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
