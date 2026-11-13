package encoder

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
)

type ProfileParams struct {
	FPS        int
	Bitrate    string
	Maxrate    string
	Bufsize    string
	GOP        int
	ScaleWidth int
}

var qualityProfiles = map[string]ProfileParams{
	"high": {
		FPS:     60,
		Bitrate: "8M",
		Maxrate: "10M",
		Bufsize: "200K",
		GOP:     60,
	},
	"medium": {
		FPS:     30,
		Bitrate: "4M",
		Maxrate: "5M",
		Bufsize: "150K",
		GOP:     30,
	},
	"low": {
		FPS:        24,
		Bitrate:    "2M",
		Maxrate:    "3M",
		Bufsize:    "100K",
		GOP:        24,
		ScaleWidth: 1280,
	},
}

func buildDDAArgs(quality string, outputIdx int) []string {
	p, ok := qualityProfiles[quality]
	if !ok {
		p = qualityProfiles["high"]
	}
	filter := fmt.Sprintf("ddagrab=output_idx=%d:framerate=%d:draw_mouse=1", outputIdx, p.FPS)
	if p.ScaleWidth > 0 {
		filter += fmt.Sprintf(",scale=%d:-2", p.ScaleWidth)
	}
	filter += ",format=yuv420p"

	return []string{
		"-filter_complex", filter,
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1", "-delay", "0",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", p.Bitrate, "-maxrate", p.Maxrate, "-bufsize", p.Bufsize,
		"-spatial-aq", "1", "-aq-strength", "8",
		"-g", fmt.Sprintf("%d", p.GOP),
		"-fflags", "nobuffer", "-flags", "low_delay", "-flush_packets", "1",
		"-f", "h264", "pipe:1",
	}
}

func buildGDIArgs(quality string, offsetX, offsetY, width, height int) []string {
	p, ok := qualityProfiles[quality]
	if !ok {
		p = qualityProfiles["high"]
	}
	if width <= 0 {
		width = 1920
	}
	if height <= 0 {
		height = 1080
	}
	width = width &^ 1
	height = height &^ 1

	args := []string{
		"-f", "gdigrab", "-framerate", fmt.Sprintf("%d", p.FPS),
		"-offset_x", fmt.Sprintf("%d", offsetX), "-offset_y", fmt.Sprintf("%d", offsetY),
		"-video_size", fmt.Sprintf("%dx%d", width, height),
		"-i", "desktop",
	}
	if p.ScaleWidth > 0 {
		args = append(args, "-vf", fmt.Sprintf("scale=%d:-2", p.ScaleWidth))
	}
	args = append(args,
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1", "-delay", "0",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", p.Bitrate, "-maxrate", p.Maxrate, "-bufsize", p.Bufsize,
		"-spatial-aq", "1", "-aq-strength", "8",
		"-g", fmt.Sprintf("%d", p.GOP),
		"-fflags", "nobuffer", "-flags", "low_delay", "-flush_packets", "1",
		"-f", "h264", "pipe:1",
	)
	return args
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

func (e *Encoder) spawn(args []string) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.Command("ffmpeg", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		return nil, nil, err
	}

	return cmd, stdout, nil
}

func (e *Encoder) StartWithMonitor(quality string, outputIdx, offsetX, offsetY, width, height int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return nil
	}

	ddaArgs := buildDDAArgs(quality, outputIdx)
	cmd, stdout, err := e.spawn(ddaArgs)
	if err == nil {
		hProc, openErr := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(cmd.Process.Pid))
		if openErr == nil {
			res, _ := syscall.WaitForSingleObject(hProc, 400)
			_ = syscall.CloseHandle(hProc)
			if res == syscall.WAIT_TIMEOUT {
				e.cmd = cmd
				e.Stdout = stdout
				e.running = true
				return nil
			}
		}
		_ = stdout.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}

	gdiArgs := buildGDIArgs(quality, offsetX, offsetY, width, height)
	cmd, stdout, err = e.spawn(gdiArgs)
	if err != nil {
		return err
	}

	e.cmd = cmd
	e.Stdout = stdout
	e.running = true
	return nil
}

func (e *Encoder) Start(quality string) error {
	return e.StartWithMonitor(quality, 0, 0, 0, 1920, 1080)
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
