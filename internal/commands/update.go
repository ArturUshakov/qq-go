package commands

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/SolasWyrd/qq-go/internal/command"
	"github.com/SolasWyrd/qq-go/internal/execx"
	"github.com/SolasWyrd/qq-go/internal/output"
	"github.com/SolasWyrd/qq-go/internal/version"
)

const (
	githubRepository       = "SolasWyrd/qq-go"
	installPath            = "/usr/local/bin/qq"
	maxDownloadSize  int64 = 100 << 20
)

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
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return fmt.Errorf("self-update не поддерживается для %s", runtime.GOOS)
	}
	if version.Version == "dev" {
		output.Warn("Текущая сборка dev. Версия будет обновлена без сравнения.")
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
	archivePath, err := downloadTempFile(assetURL, assetName)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath)
	checksumURL := "https://github.com/" + githubRepository + "/releases/download/" + release.TagName + "/checksums.txt"
	if err := verifyReleaseChecksum(archivePath, assetName, checksumURL); err != nil {
		return err
	}
	binaryPath, err := extractBinary(archivePath)
	if err != nil {
		return err
	}
	defer os.Remove(binaryPath)
	if err := installBinary(binaryPath); err != nil {
		return err
	}
	output.Success("qq %s установлен: %s", release.TagName, installPath)
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

func downloadTempFile(url string, assetName string) (string, error) {
	client := &http.Client{Timeout: 2 * time.Minute}
	response, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("не удалось скачать архив: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("не удалось скачать архив: %s", response.Status)
	}
	if response.ContentLength > maxDownloadSize {
		return "", fmt.Errorf("архив превышает лимит %d MB", maxDownloadSize>>20)
	}
	file, err := os.CreateTemp("", "qq-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	progress := output.NewProgress("Скачивание "+assetName, response.ContentLength)
	written, err := io.Copy(file, io.LimitReader(progress.Wrap(response.Body), maxDownloadSize+1))
	if err != nil {
		return "", fmt.Errorf("ошибка загрузки архива: %w", err)
	}
	if written > maxDownloadSize {
		return "", fmt.Errorf("архив превышает лимит %d MB", maxDownloadSize>>20)
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	progress.Finish("Архив скачан")
	return file.Name(), nil
}

func verifyReleaseChecksum(archivePath, assetName, checksumURL string) error {
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("не удалось скачать checksums.txt: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums.txt недоступен: %s", response.Status)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	expected := ""
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.TrimPrefix(fields[len(fields)-1], "*") == assetName {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum для %s не найден", assetName)
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum архива не совпадает")
	}
	output.Success("SHA-256 проверен")
	return nil
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

func installBinary(sourcePath string) error {
	if err := installBinaryDirect(sourcePath); err == nil {
		return nil
	}
	if !execx.Exists("sudo") {
		return fmt.Errorf("нет прав на запись в %s и sudo не найден", installPath)
	}
	if err := execx.RunPassthrough("sudo", "install", "-m", "0755", sourcePath, installPath); err != nil {
		return fmt.Errorf("не удалось установить бинарник через sudo: %w", err)
	}
	return nil
}

func installBinaryDirect(sourcePath string) error {
	directory := filepath.Dir(installPath)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	input, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer input.Close()
	temp, err := os.CreateTemp(directory, ".qq-update-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := io.Copy(temp, input); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(0755); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, installPath); err != nil {
		return err
	}
	return nil
}
