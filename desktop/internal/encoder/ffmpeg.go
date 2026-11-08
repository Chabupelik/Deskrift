package encoder

import (
	"io"
	"os/exec"
	"sync"
	"syscall"
)

var QualityProfiles = map[string][]string{
	"high": {
		"-f", "gdigrab", "-framerate", "60", "-i", "desktop",
		"-c:v", "h264_nvenc", "-preset", "p1", "-tune", "ll",
		"-zerolatency", "1", "-rc", "cbr",
		"-b:v", "15M", "-maxrate", "15M", "-bufsize", "3M",
		"-g", "60",
		"-f", "h264", "pipe:1",
	},
	"medium": {
		"-f", "gdigrab", "-framerate", "30", "-i", "desktop",
		"-c:v", "h264_nvenc", "-preset", "p1", "-tune", "ll",
		"-zerolatency", "1", "-rc", "cbr",
		"-b:v", "6M", "-maxrate", "6M", "-bufsize", "2M",
		"-g", "30",
		"-f", "h264", "pipe:1",
	},
	"low": {
		"-f", "gdigrab", "-framerate", "24", "-i", "desktop",
		"-vf", "scale=1280:720",
		"-c:v", "h264_nvenc", "-preset", "p1", "-tune", "ll",
		"-zerolatency", "1", "-rc", "cbr",
		"-b:v", "2M", "-maxrate", "2M", "-bufsize", "1M",
		"-g", "24",
		"-f", "h264", "pipe:1",
	},
}

type Encoder struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	Stdout  io.ReadCloser
	running bool
}

func New() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Start(quality string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return nil
	}

	args, ok := QualityProfiles[quality]
	if !ok {
		args = QualityProfiles["high"]
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	e.cmd = cmd
	e.Stdout = stdout
	e.running = true
	return nil
}

func (e *Encoder) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	if e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
		_ = e.cmd.Wait()
	}

	if e.Stdout != nil {
		_ = e.Stdout.Close()
	}

	e.running = false
	e.cmd = nil
	e.Stdout = nil
}

func (e *Encoder) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}
