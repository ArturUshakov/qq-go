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

func TestFormatPublishedPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		container string
		service   string
		image     string
		mapping   portMapping
		want      string
	}{
		{
			name:      "nginx is http link",
			container: "hr-nginx",
			service:   "nginx",
			image:     "nginx:alpine",
			mapping:   portMapping{hostPort: "8185", containerPort: "80"},
			want:      "http://localhost:8185",
		},
		{
			name:      "https target",
			container: "proxy",
			service:   "traefik",
			image:     "traefik:v3",
			mapping:   portMapping{hostPort: "8443", containerPort: "443"},
			want:      "https://localhost:8443",
		},
		{
			name:      "redis is plain address",
			container: "hr-redis",
			service:   "redis",
			image:     "redis:7",
			mapping:   portMapping{hostPort: "6379", containerPort: "6379"},
			want:      "localhost:6379",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := formatPublishedPort(test.container, test.service, test.image, test.mapping)
			if got != test.want {
				t.Fatalf("formatPublishedPort() = %q, want %q", got, test.want)
			}
		})
	}
}
