package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Progress struct {
	label       string
	total       int64
	current     int64
	startedAt   time.Time
	lastRender  time.Time
	lastPercent int
	interactive bool
	mu          sync.Mutex
}

func NewProgress(label string, total int64) *Progress {
	progress := &Progress{label: label, total: total, startedAt: time.Now(), lastPercent: -1, interactive: IsTerminal(os.Stdout)}
	if progress.interactive {
		progress.render(true)
	} else {
		Plain("[START] %s (%s)", label, formatBytes(total))
	}
	return progress
}

func (progress *Progress) Wrap(reader io.Reader) io.Reader {
	return &progressReader{reader: reader, progress: progress}
}

func (progress *Progress) Finish(message string) {
	progress.mu.Lock()
	defer progress.mu.Unlock()
	progress.current = progress.total
	if progress.interactive {
		fmt.Fprint(os.Stdout, "\r\033[2K")
		Success("DONE  %s (%s)", message, formatDuration(time.Since(progress.startedAt)))
		return
	}
	Plain("[DONE] %s (%s)", message, formatDuration(time.Since(progress.startedAt)))
}

func (progress *Progress) advance(count int64) {
	progress.mu.Lock()
	defer progress.mu.Unlock()
	progress.current += count
	progress.render(false)
}

func (progress *Progress) render(force bool) {
	if progress.total <= 0 {
		return
	}
	percent := int(progress.current * 100 / progress.total)
	if percent > 100 {
		percent = 100
	}
	if progress.interactive {
		if !force && time.Since(progress.lastRender) < 100*time.Millisecond && progress.current < progress.total {
			return
		}
		progress.lastRender = time.Now()
		width := 30
		filled := width * percent / 100
		bar := strings.Repeat("=", filled)
		if filled < width {
			bar += ">" + strings.Repeat(" ", width-filled-1)
		}
		elapsed := time.Since(progress.startedAt)
		rate := float64(progress.current) / elapsed.Seconds()
		eta := time.Duration(0)
		if rate > 0 && progress.current < progress.total {
			eta = time.Duration(float64(progress.total-progress.current)/rate) * time.Second
		}
		fmt.Fprintf(os.Stdout, "\r\033[2K%-24s [%s] %3d%%  %s/%s  %s/s  ETA %s", progress.label, bar, percent, formatBytes(progress.current), formatBytes(progress.total), formatBytes(int64(rate)), formatDuration(eta))
		return
	}
	if percent == progress.lastPercent || (percent%10 != 0 && progress.current < progress.total) {
		return
	}
	progress.lastPercent = percent
	Plain("[%3d%%] %s/%s | elapsed %s", percent, formatBytes(progress.current), formatBytes(progress.total), formatDuration(time.Since(progress.startedAt)))
}

type progressReader struct {
	reader   io.Reader
	progress *Progress
}

func (reader *progressReader) Read(buffer []byte) (int, error) {
	count, err := reader.reader.Read(buffer)
	if count > 0 {
		reader.progress.advance(int64(count))
	}
	return count, err
}

func formatBytes(value int64) string {
	if value < 0 {
		return "unknown"
	}
	units := []string{"B", "KB", "MB", "GB"}
	amount := float64(value)
	unit := 0
	for amount >= 1024 && unit < len(units)-1 {
		amount /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", value, units[unit])
	}
	return fmt.Sprintf("%.1f %s", amount, units[unit])
}
func formatDuration(value time.Duration) string {
	if value < 10*time.Second {
		return fmt.Sprintf("%.1fs", value.Seconds())
	}
	return value.Round(time.Second).String()
}
