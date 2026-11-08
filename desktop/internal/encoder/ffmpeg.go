package encoder

import (
	"io"
	"log"
	"os/exec"

	"github.com/chabupelik/deskrift/internal/config"
)

var QualityProfiles = map[string][]string{
	"high": {
		"-f", "gdigrab", "-framerate", "60", "-i", "desktop",
		"-c:v", "h264_nvenc", "-preset", "p1", "-tune", "ll",
		"-b:v", "15M", "-maxrate", "15M", "-bufsize", "5M",
		"-g", "60",
		"-f", "h264", "pipe:1",
	},
	"medium": {
		"-f", "gdigrab", "-framerate", "30", "-i", "desktop",
		"-c:v", "h264_nvenc", "-preset", "p1", "-tune", "ll",
		"-b:v", "6M", "-maxrate", "6M", "-bufsize", "2M",
		"-g", "30",
		"-f", "h264", "pipe:1",
	},
	"low": {
		"-f", "gdigrab", "-framerate", "24", "-i", "desktop",
		"-vf", "scale=1280:720",
		"-c:v", "h264_nvenc", "-preset", "p1", "-tune", "ll",
		"-b:v", "2M", "-maxrate", "2M", "-bufsize", "1M",
		"-g", "24",
		"-f", "h264", "pipe:1",
	},
}

type Encoder struct {
	cfg    *config.Config
	cmd    *exec.Cmd
	Stdout io.ReadCloser
}

func New(cfg *config.Config) *Encoder {
	return &Encoder{cfg: cfg}
}

func (e *Encoder) Start() error {
	args, ok := QualityProfiles[e.cfg.Quality]
	if !ok {
		args = QualityProfiles["high"]
	}

	e.cmd = exec.Command("ffmpeg", args...)
	var err error
	e.Stdout, err = e.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := e.cmd.Start(); err != nil {
		return err
	}
	log.Printf("ffmpeg started (quality: %s)", e.cfg.Quality)
	return nil
}

func (e *Encoder) Stop() {
	if e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
	}
}

func (e *Encoder) SetQuality(quality string) error {
	log.Printf("Switching quality to: %s", quality)
	e.Stop()
	e.cfg.Quality = quality
	return e.Start()
}
