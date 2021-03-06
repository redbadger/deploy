package agent

import (
	"reflect"
	"testing"
)

func Test_formatResults(t *testing.T) {
	type args struct {
		in map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantOut string
	}{
		{
			"one key",
			args{
				map[string]string{"one": "test"},
			},
			"* one:\n\ttest\n\n",
		},
		{
			"two keys",
			args{
				map[string]string{"two": "test2", "one": "test1"},
			},
			"* one:\n\ttest1\n\n* two:\n\ttest2\n\n",
		},
		{
			"two key multiline",
			args{
				map[string]string{"one": "test1a\ntest1b", "two": "test2a\ntest2b"},
			},
			"* one:\n\ttest1a\n\ttest1b\n\n* two:\n\ttest2a\n\ttest2b\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOut := formatResults(tt.args.in); gotOut != tt.wantOut {
				t.Errorf("formatResults() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}

func Test_keys(t *testing.T) {
	type args struct {
		m map[string]string
	}
	tests := []struct {
		name     string
		args     args
		wantKeys []string
	}{
		{
			"gets sorted keys",
			args{
				map[string]string{"b": "2", "a": "1"},
			},
			[]string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotKeys := keys(tt.args.m); !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Errorf("keys() = %v, want %v", gotKeys, tt.wantKeys)
			}
		})
	}
}

func Test_joinManifests(t *testing.T) {
	type args struct {
		namespace string
		manifests []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"joins manifests and puts namespace manifest in front",
			args{
				"my-namespace",
				[]string{"a\n", "b\n"},
			},
			"---\napiVersion: v1\nkind: Namespace\nmetadata:\n  name: my-namespace\n---\na\n\n---\nb\n",
		},
		{
			"manifests don't have final newlines",
			args{
				"my-namespace",
				[]string{"a", "b"},
			},
			"---\napiVersion: v1\nkind: Namespace\nmetadata:\n  name: my-namespace\n---\na\n---\nb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinManifests(tt.args.namespace, tt.args.manifests); got != tt.want {
				t.Errorf("joinManifests() = %v, want %v", got, tt.want)
			}
		})
	}
}
