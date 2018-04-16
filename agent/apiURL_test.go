package agent

import (
	"testing"
)

func TestRootAPI(t *testing.T) {
	type args struct {
		repoURL string
	}
	tests := []struct {
		name       string
		args       args
		wantAPIURL string
		wantErr    bool
	}{
		{
			"public github",
			args{"https://api.github.com/repos/my-org/my-repo/pulls/1"},
			"https://api.github.com",
			false,
		},
		{
			"enterprise github",
			args{"https://github.my-domain/api/v3/users/me"},
			"https://github.my-domain/api/v3",
			false,
		},
		{
			"not an API URL",
			args{"https://github.my-domain/api/v999/users/me"},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAPIURL, err := RootAPI(tt.args.repoURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("RootAPI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotAPIURL != tt.wantAPIURL {
				t.Errorf("RootAPI() = %v, want %v", gotAPIURL, tt.wantAPIURL)
			}
		})
	}
}
