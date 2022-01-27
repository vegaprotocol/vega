package validation_test

import (
	"testing"

	"code.vegaprotocol.io/vega/oracles/validation"
)

func TestCheckForInternalOracle(t *testing.T) {
	type args struct {
		data map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should return an error if there is any data that contains reserved prefix",
			args: args{
				data: map[string]string{
					"aaaa":                      "aaaa",
					"bbbb":                      "bbbb",
					"cccc":                      "cccc",
					"vegaprotocol.builtin.dddd": "dddd",
				},
			},
			wantErr: true,
		},
		{
			name: "Should pass validation if none of the data contains a reserved prefix",
			args: args{
				data: map[string]string{
					"aaaa": "aaaa",
					"bbbb": "bbbb",
					"cccc": "cccc",
					"dddd": "dddd",
				},
			},
			wantErr: false,
		},
		{
			name: "Should pass validation if reserved prefix is contained in key, but key doesn't start with the prefix",
			args: args{
				data: map[string]string{
					"aaaa":                      "aaaa",
					"bbbb":                      "bbbb",
					"cccc":                      "cccc",
					"dddd.vegaprotocol.builtin": "dddd",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validation.CheckForInternalOracle(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("CheckForInternalOracle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
