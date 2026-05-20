package commands

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ArturUshakov/qq-go/internal/command"
	"github.com/ArturUshakov/qq-go/internal/execx"
	"github.com/ArturUshakov/qq-go/internal/output"
)

func RegisterContainers(registry *command.Registry) error {
	commands := []command.Command{
		{Names: []string{"list", "-l"}, Group: "container", Description: "Показать запущенные Docker-проекты", Run: listContainers},
		{Names: []string{"down", "-d"}, Group: "container", Description: "Остановить все контейнеры или контейнеры по имени/проекту", Run: stopContainers},
		{Names: []string{"exec", "-e"}, Group: "container", Description: "Войти в контейнер по части имени. Флаг -r — root", Run: execInContainer},
		{Names: []string{"network", "-net"}, Group: "container", Description: "Показать локальный IP и проверить HTTP-доступ к портам контейнеров", Run: networkConnect},
	}
	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func listContainers(args []string) error {
	format := `{{.Names}}	{{.Status}}	{{.Label "com.docker.compose.project"}}	{{.Image}}	{{.Ports}}`
	result, err := execx.Output("docker", "ps", "-a", "--format", format)
	if err != nil {
		return err
	}
	lines := splitLines(result.Stdout)
	if len(lines) == 0 {
		output.Info("Нет контейнеров")
		return nil
	}
	webServices := []string{"nginx", "apache", "node", "vite", "react", "flask", "laravel", "express", "next", "nuxt", "adminer", "grafana", "portainer"}
	type containerInfo struct{ name, status, portInfo string }
	allProjects := make(map[string][]containerInfo)
	activeProjects := make(map[string]bool)
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		name, status, project, image := parts[0], parts[1], parts[2], parts[3]
		ports := ""
		if len(parts) >= 5 {
			ports = parts[4]
		}
		if project == "" {
			project = "без docker-compose project"
		}
		statusRU := translateStatus(status)
		portInfo := ports
		if strings.Contains(status, "Up") {
			urlPorts := extractPorts(ports)
			formattedPorts := make([]string, 0, len(urlPorts))
			for _, port := range urlPorts {
				if containsService(image, webServices) {
					formattedPorts = append(formattedPorts, "http://localhost:"+port)
				} else {
					formattedPorts = append(formattedPorts, port+"/tcp")
				}
			}
			if len(formattedPorts) > 0 {
				portInfo = strings.Join(uniqueSorted(formattedPorts), ", ")
			}
			activeProjects[project] = true
		}
		allProjects[project] = append(allProjects[project], containerInfo{name: name, status: statusRU, portInfo: portInfo})
	}
	if len(activeProjects) == 0 {
		output.Info("Нет активных проектов")
		return nil
	}
	projects := make([]string, 0, len(activeProjects))
	for project := range activeProjects {
		projects = append(projects, project)
	}
	sort.Strings(projects)
	for _, project := range projects {
		output.Title("\nПроект: %s", project)
		output.Plain(strings.Repeat("─", 90))
		for _, item := range allProjects[project] {
			fmt.Printf("%-36s │ %-14s │ %s\n", item.name, item.status, item.portInfo)
		}
	}
	return nil
}

func stopContainers(args []string) error {
	startedAt := time.Now()
	if len(args) == 0 {
		result, err := execx.Output("docker", "ps", "-q")
		if err != nil {
			return err
		}
		ids := splitLines(result.Stdout)
		if len(ids) == 0 {
			output.Warn("Нет запущенных контейнеров для остановки")
			return nil
		}
		killArgs := append([]string{"kill"}, ids...)
		if err := execx.RunPassthrough("docker", killArgs...); err != nil {
			return err
		}
		output.Success("Все контейнеры остановлены")
		output.Plain("Время выполнения: %.2f секунд", time.Since(startedAt).Seconds())
		return nil
	}
	filter := args[0]
	format := `{{.ID}}	{{.Names}}	{{.Label "com.docker.compose.project"}}`
	result, err := execx.Output("docker", "ps", "--format", format)
	if err != nil {
		return err
	}
	var ids []string
	for _, line := range splitLines(result.Stdout) {
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		project := ""
		if len(parts) >= 3 {
			project = parts[2]
		}
		if strings.Contains(project, filter) || strings.Contains(parts[1], filter) {
			ids = append(ids, parts[0])
		}
	}
	if len(ids) == 0 {
		output.Warn("Контейнеры по фильтру %q не найдены", filter)
		return nil
	}
	killArgs := append([]string{"kill"}, ids...)
	if err := execx.RunPassthrough("docker", killArgs...); err != nil {
		return err
	}
	output.Success("Контейнеры остановлены: %d", len(ids))
	output.Plain("Время выполнения: %.2f секунд", time.Since(startedAt).Seconds())
	return nil
}

func execInContainer(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("укажите часть имени контейнера")
	}
	asRoot := false
	partialName := ""
	commandArgs := make([]string, 0)
	for _, arg := range args {
		if arg == "-r" || arg == "--root" {
			asRoot = true
			continue
		}
		if partialName == "" {
			partialName = arg
			continue
		}
		commandArgs = append(commandArgs, arg)
	}
	if partialName == "" {
		return fmt.Errorf("укажите часть имени контейнера")
	}
	containerName, err := findContainer(partialName)
	if err != nil {
		return err
	}
	if len(commandArgs) == 0 {
		commandArgs = []string{"bash"}
		if _, err := execx.Output("docker", "exec", containerName, "which", "bash"); err != nil {
			commandArgs = []string{"sh"}
		}
	}
	dockerArgs := []string{"exec", "-it"}
	if asRoot {
		dockerArgs = append(dockerArgs, "--user", "root")
	}
	dockerArgs = append(dockerArgs, containerName)
	dockerArgs = append(dockerArgs, commandArgs...)
	output.Success("Вход в контейнер: %s", containerName)
	return execx.RunInteractive("docker", dockerArgs...)
}

func networkConnect(args []string) error {
	ip := getLocalIP()
	result, err := execx.Output("docker", "ps", "--format", `{{.Names}}	{{.Ports}}`)
	if err != nil {
		return err
	}
	type option struct{ name, port string }
	options := make([]option, 0)
	seen := make(map[string]struct{})
	for _, line := range splitLines(result.Stdout) {
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}
		for _, port := range extractPorts(parts[1]) {
			key := parts[0] + ":" + port
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			options = append(options, option{name: parts[0], port: port})
		}
	}
	if len(options) == 0 {
		output.Warn("Нет контейнеров с проброшенными портами")
		return nil
	}
	output.Info("Локальный IP вашей машины: %s", ip)
	for i, option := range options {
		output.Plain("%d. %-35s → http://%s:%s", i+1, option.name, ip, option.port)
	}
	fmt.Print("\nВведите номер проекта: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("не удалось прочитать ввод")
	}
	choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || choice < 1 || choice > len(options) {
		return fmt.Errorf("неверный ввод")
	}
	selected := options[choice-1]
	url := fmt.Sprintf("http://%s:%s", ip, selected.port)
	if checkHTTP(url) {
		output.Success("Соединение успешно: %s", url)
		return nil
	}
	return fmt.Errorf("не удалось подключиться к %s", url)
}

func findContainer(partialName string) (string, error) {
	result, err := execx.Output("docker", "ps", "--format", "{{.Names}}")
	if err != nil {
		return "", err
	}
	partial := strings.ToLower(partialName)
	matches := make([]string, 0)
	for _, name := range splitLines(result.Stdout) {
		if strings.Contains(strings.ToLower(name), partial) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("контейнер с частью имени %q не найден", partialName)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	output.Warn("Найдено несколько контейнеров:")
	for i, name := range matches {
		output.Plain("%d. %s", i+1, name)
	}
	fmt.Print("Введите номер контейнера: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("не удалось прочитать ввод")
	}
	choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || choice < 1 || choice > len(matches) {
		return "", fmt.Errorf("неверный выбор")
	}
	return matches[choice-1], nil
}

func translateStatus(status string) string {
	switch {
	case strings.Contains(status, "Up"):
		return "Запущен"
	case strings.Contains(status, "Exited"):
		return "Остановлен"
	case strings.Contains(status, "Created"):
		return "Создан"
	case strings.Contains(status, "Restarting"):
		return "Перезапуск"
	default:
		return status
	}
}

func extractPorts(ports string) []string {
	regexpValue := regexp.MustCompile(`(?:[\d.]+:)?(\d+)->|^(\d+)/tcp$`)
	matches := regexpValue.FindAllStringSubmatch(ports, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		port := match[1]
		if port == "" && len(match) > 2 {
			port = match[2]
		}
		if port != "" {
			result = append(result, port)
		}
	}
	return result
}

func containsService(image string, services []string) bool {
	image = strings.ToLower(image)
	for _, service := range services {
		if strings.Contains(image, service) {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
