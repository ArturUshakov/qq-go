package commands

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ArturUshakov/qq-go/internal/command"
	"github.com/ArturUshakov/qq-go/internal/execx"
	"github.com/ArturUshakov/qq-go/internal/output"
	"github.com/ArturUshakov/qq-go/internal/version"
)

const githubRepository = "ArturUshakov/qq-go"

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func updateCommand(registry *command.Registry) command.Command {
	return command.Command{
		Names:       []string{"update", "self-update"},
		Group:       "base",
		Description: "Обновляет qq до последнего GitHub Release",
		Run: func(args []string) error {
			return selfUpdate(registry)
		},
	}
}

func selfUpdate(registry *command.Registry) error {
	if version.Version == "dev" {
		output.Warn("Текущая сборка dev. Проверка версии будет выполнена без сравнения.")
	}
	release, err := fetchLatestRelease()
	if err != nil {
		return err
	}
	if version.Version != "dev" && strings.TrimPrefix(release.TagName, "v") == strings.TrimPrefix(version.Version, "v") {
		output.Success("Установлена актуальная версия: %s", version.Version)
		return nil
	}
	assetURL, assetName, err := findReleaseAsset(release)
	if err != nil {
		return err
	}
	output.Info("Найдена версия %s: %s", release.TagName, assetName)
	archivePath, err := downloadTempFile(assetURL)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath)
	binaryPath, err := extractBinary(archivePath)
	if err != nil {
		return err
	}
	defer os.Remove(binaryPath)
	currentPath, err := os.Executable()
	if err != nil {
		return err
	}
	resolvedPath, err := filepath.EvalSymlinks(currentPath)
	if err != nil {
		resolvedPath = currentPath
	}
	if err := replaceBinary(resolvedPath, binaryPath); err != nil {
		return err
	}
	output.Success("qq обновлен до %s", release.TagName)
	updateCompletionAfterSelfUpdate(registry)
	return nil
}

func updateCompletionAfterSelfUpdate(registry *command.Registry) {
	shell := detectShell()
	if shell == "" {
		output.Warn("Не удалось определить shell для обновления completion. Выполните вручную: qq completion install")
		return
	}
	if err := installCompletion(registry, shell); err != nil {
		output.Warn("Не удалось обновить completion: %s", err.Error())
		output.Info("Выполните вручную: qq completion install %s", shell)
	}
}

func fetchLatestRelease() (githubRelease, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	request, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+githubRepository+"/releases/latest", nil)
	if err != nil {
		return githubRelease{}, err
	}
	request.Header.Set("User-Agent", "qq-cli")
	if token := githubToken(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return fetchLatestReleaseFromRedirect()
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
			return fetchLatestReleaseFromRedirect()
		}
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		apiErr := fmt.Errorf("GitHub API вернул %s: %s", response.Status, strings.TrimSpace(string(body)))
		release, redirectErr := fetchLatestReleaseFromRedirect()
		if redirectErr == nil {
			output.Warn("GitHub API недоступен (%s), используется fallback через releases/latest", response.Status)
			return release, nil
		}
		return githubRelease{}, apiErr
	}
	var release githubRelease
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return githubRelease{}, err
	}
	return release, nil
}

func githubToken() string {
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("GH_TOKEN"))
}

func fetchLatestReleaseFromRedirect() (githubRelease, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	request, err := http.NewRequest(http.MethodGet, "https://github.com/"+githubRepository+"/releases/latest", nil)
	if err != nil {
		return githubRelease{}, err
	}
	request.Header.Set("User-Agent", "qq-cli")
	response, err := client.Do(request)
	if err != nil {
		return githubRelease{}, fmt.Errorf("не удалось определить latest release через GitHub redirect: %w", err)
	}
	defer response.Body.Close()

	location := response.Header.Get("Location")
	if location == "" {
		return githubRelease{}, fmt.Errorf("GitHub releases/latest не вернул redirect")
	}
	tag := strings.TrimSpace(pathBase(location))
	if tag == "" || tag == "latest" {
		return githubRelease{}, fmt.Errorf("не удалось извлечь tag из redirect: %s", location)
	}
	return githubRelease{TagName: tag}, nil
}

func pathBase(value string) string {
	value = strings.TrimRight(value, "/")
	index := strings.LastIndex(value, "/")
	if index == -1 {
		return value
	}
	return value[index+1:]
}

func findReleaseAsset(release githubRelease) (string, string, error) {
	assetPart := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	assetName := "qq_" + assetPart + ".tar.gz"
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, assetPart) && strings.HasSuffix(asset.Name, ".tar.gz") {
			return asset.BrowserDownloadURL, asset.Name, nil
		}
	}
	if release.TagName != "" {
		return "https://github.com/" + githubRepository + "/releases/download/" + release.TagName + "/" + assetName, assetName, nil
	}
	return "", "", fmt.Errorf("не найден release-asset для %s/%s", runtime.GOOS, runtime.GOARCH)
}

func downloadTempFile(url string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("не удалось скачать архив: %s", response.Status)
	}
	file, err := os.CreateTemp("", "qq-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.Copy(file, response.Body); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func extractBinary(archivePath string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(header.Name) != "qq" {
			continue
		}
		binary, err := os.CreateTemp("", "qq-bin-*")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(binary, tarReader); err != nil {
			binary.Close()
			return "", err
		}
		if err := binary.Close(); err != nil {
			return "", err
		}
		if err := os.Chmod(binary.Name(), 0755); err != nil {
			return "", err
		}
		return binary.Name(), nil
	}
	return "", fmt.Errorf("в архиве не найден бинарник qq")
}

func replaceBinary(targetPath string, sourcePath string) error {
	backupPath := targetPath + ".backup"
	_ = os.Remove(backupPath)
	if err := os.Rename(targetPath, backupPath); err != nil {
		return fmt.Errorf("не удалось создать backup текущего бинарника: %w", err)
	}
	input, err := os.Open(sourcePath)
	if err != nil {
		_ = os.Rename(backupPath, targetPath)
		return err
	}
	defer input.Close()
	outputFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		_ = os.Rename(backupPath, targetPath)
		return err
	}
	if _, err := io.Copy(outputFile, input); err != nil {
		outputFile.Close()
		_ = os.Rename(backupPath, targetPath)
		return err
	}
	if err := outputFile.Close(); err != nil {
		_ = os.Rename(backupPath, targetPath)
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}

func checkTool(name string) {
	if execx.Exists(name) {
		output.Success("✔ %s найден", name)
		return
	}
	output.Warn("⚠ %s не найден", name)
}
