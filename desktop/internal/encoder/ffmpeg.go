package encoder

import (
	"io"
	"os/exec"
	"sync"
	"syscall"
)

var ddaProfiles = map[string][]string{
	"high": {
		"-filter_complex", "ddagrab=output_idx=0:framerate=60:draw_mouse=1,format=yuv420p",
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1", "-delay", "0",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", "6M", "-maxrate", "8M", "-bufsize", "500K",
		"-g", "60",
		"-fflags", "nobuffer", "-flags", "low_delay", "-flush_packets", "1",
		"-f", "h264", "pipe:1",
	},
	"medium": {
		"-filter_complex", "ddagrab=output_idx=0:framerate=30:draw_mouse=1,format=yuv420p",
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1", "-delay", "0",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", "3M", "-maxrate", "4M", "-bufsize", "300K",
		"-g", "30",
		"-fflags", "nobuffer", "-flags", "low_delay", "-flush_packets", "1",
		"-f", "h264", "pipe:1",
	},
	"low": {
		"-filter_complex", "ddagrab=output_idx=0:framerate=24:draw_mouse=1,scale=1280:720,format=yuv420p",
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1", "-delay", "0",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", "1500K", "-maxrate", "2M", "-bufsize", "200K",
		"-g", "24",
		"-fflags", "nobuffer", "-flags", "low_delay", "-flush_packets", "1",
		"-f", "h264", "pipe:1",
	},
}

var gdiProfiles = map[string][]string{
	"high": {
		"-f", "gdigrab", "-framerate", "60",
		"-offset_x", "0", "-offset_y", "0", "-video_size", "1920x1080",
		"-i", "desktop",
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1", "-delay", "0",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", "6M", "-maxrate", "8M", "-bufsize", "500K",
		"-g", "60",
		"-fflags", "nobuffer", "-flags", "low_delay", "-flush_packets", "1",
		"-f", "h264", "pipe:1",
	},
	"medium": {
		"-f", "gdigrab", "-framerate", "30",
		"-offset_x", "0", "-offset_y", "0", "-video_size", "1920x1080",
		"-i", "desktop",
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1", "-delay", "0",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", "3M", "-maxrate", "4M", "-bufsize", "300K",
		"-g", "30",
		"-fflags", "nobuffer", "-flags", "low_delay", "-flush_packets", "1",
		"-f", "h264", "pipe:1",
	},
	"low": {
		"-f", "gdigrab", "-framerate", "24",
		"-offset_x", "0", "-offset_y", "0", "-video_size", "1920x1080",
		"-i", "desktop",
		"-vf", "scale=1280:720",
		"-c:v", "h264_nvenc",
		"-preset", "p1", "-tune", "ull", "-zerolatency", "1", "-delay", "0",
		"-aud", "1", "-forced-idr", "1",
		"-profile:v", "baseline",
		"-bf", "0", "-b_ref_mode", "0",
		"-b:v", "1500K", "-maxrate", "2M", "-bufsize", "200K",
		"-g", "24",
		"-fflags", "nobuffer", "-flags", "low_delay", "-flush_packets", "1",
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

func (e *Encoder) Start(quality string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return nil
	}

	ddaArgs, ok := ddaProfiles[quality]
	if !ok {
		ddaArgs = ddaProfiles["high"]
	}

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

	gdiArgs, ok := gdiProfiles[quality]
	if !ok {
		gdiArgs = gdiProfiles["high"]
	}

	cmd, stdout, err = e.spawn(gdiArgs)
	if err != nil {
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
