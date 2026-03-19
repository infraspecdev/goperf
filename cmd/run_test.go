package cmd

import (
	"testing"
	"time"
)

func TestFlagRegistration(t *testing.T) {
	tests := []struct {
		name      string
		shorthand string
		wantDef   interface{}
	}{
		{"requests", "n", 1},
		{"timeout", "t", 10 * time.Second},
		{"concurrency", "c", 1},
		{"duration", "d", 0 * time.Second},
		{"method", "m", "GET"},
		{"body", "b", ""},
		{"header", "H", []string{}},
		{"config", "f", ""},
		{"verbose", "v", false},
	}

	cmd := newRunCmd()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.name)
			if flag == nil {
				t.Fatalf("expected flag --%s to exist", tt.name)
			}

			if flag.Shorthand != tt.shorthand {
				t.Errorf("expected shorthand -%s, got -%s", tt.shorthand, flag.Shorthand)
			}

			var got interface{}
			var err error
			switch tt.wantDef.(type) {
			case int:
				got, err = cmd.Flags().GetInt(tt.name)
			case time.Duration:
				got, err = cmd.Flags().GetDuration(tt.name)
			case string:
				got, err = cmd.Flags().GetString(tt.name)
			case []string:
				got, err = cmd.Flags().GetStringArray(tt.name)
			case bool:
				got, err = cmd.Flags().GetBool(tt.name)
			default:
				t.Fatalf("unsupported flag type for %s", tt.name)
			}

			if err != nil {
				t.Fatalf("error getting flag %s: %v", tt.name, err)
			}

			if wantArr, ok := tt.wantDef.([]string); ok {
				gotArr := got.([]string)
				if len(wantArr) != len(gotArr) {
					t.Errorf("expected default %v, got %v", wantArr, gotArr)
				}
				for i := range wantArr {
					if wantArr[i] != gotArr[i] {
						t.Errorf("expected default %v, got %v", wantArr, gotArr)
						break
					}
				}
				return
			}

			if got != tt.wantDef {
				t.Errorf("expected default %v, got %v", tt.wantDef, got)
			}
		})
	}
}
