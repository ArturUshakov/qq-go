package commands

import (
	"reflect"
	"testing"
)

func TestExtractPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "IPv4", input: "0.0.0.0:8080->80/tcp", want: []string{"8080"}},
		{name: "IPv6", input: ":::8443->443/tcp", want: []string{"8443"}},
		{name: "multiple", input: "0.0.0.0:8080->80/tcp, [::]:8080->80/tcp, 127.0.0.1:5432->5432/tcp", want: []string{"5432", "8080"}},
		{name: "not published", input: "80/tcp", want: []string{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := extractPorts(test.input); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("extractPorts(%q) = %#v, want %#v", test.input, got, test.want)
			}
		})
	}
}

func TestTranslateStatus(t *testing.T) {
	t.Parallel()
	if got := translateStatus("Up 2 minutes"); got != "Запущен" {
		t.Fatalf("translateStatus() = %q", got)
	}
}
