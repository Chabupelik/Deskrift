package encoder

import (
	"io"
	"os/exec"
	"sync"
	"syscall"
)

var QualityProfiles = map[string][]string{
	"high": {
		"-f", "gdigrab", "-framerate", "60",
		"-offset_x", "0", "-offset_y", "0", "-video_size", "1920x1080",
		"-i", "desktop",
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", "15M", "-maxrate", "15M", "-bufsize", "2M",
		"-g", "60",
		"-f", "h264", "pipe:1",
	},
	"medium": {
		"-f", "gdigrab", "-framerate", "30",
		"-offset_x", "0", "-offset_y", "0", "-video_size", "1920x1080",
		"-i", "desktop",
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", "6M", "-maxrate", "6M", "-bufsize", "1M",
		"-g", "30",
		"-f", "h264", "pipe:1",
	},
	"low": {
		"-f", "gdigrab", "-framerate", "24",
		"-offset_x", "0", "-offset_y", "0", "-video_size", "1920x1080",
		"-i", "desktop",
		"-vf", "scale=1280:720",
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", "2M", "-maxrate", "2M", "-bufsize", "500K",
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
