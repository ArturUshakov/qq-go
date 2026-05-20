package commands

import (
	"net"
	"net/http"
	"strings"
	"time"
)

func splitLines(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func getLocalIP() string {
	connection, err := net.Dial("udp", "10.255.255.255:1")
	if err != nil {
		return "127.0.0.1"
	}
	defer connection.Close()
	localAddress, ok := connection.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "127.0.0.1"
	}
	return localAddress.IP.String()
}

func checkHTTP(url string) bool {
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode >= 200 && response.StatusCode < 400
}
