package relay

import (
	"io"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/chabupelik/deskrift/internal/config"
	"github.com/chabupelik/deskrift/internal/encoder"
	"github.com/chabupelik/deskrift/internal/input"
)

const (
	chunkSize      = 32 * 1024 // 32KB chunks
	reconnectDelay = 3 * time.Second
)

type Client struct {
	cfg     *config.Config
	enc     *encoder.Encoder
	input   *input.Handler
	conn    *websocket.Conn
	done    chan struct{}
}

func New(cfg *config.Config, enc *encoder.Encoder, inp *input.Handler) *Client {
	return &Client{
		cfg:   cfg,
		enc:   enc,
		input: inp,
		done:  make(chan struct{}),
	}
}

func (c *Client) Run() error {
	for {
		select {
		case <-c.done:
			return nil
		default:
		}

		if err := c.connect(); err != nil {
			log.Printf("connection failed: %v, retrying in %v", err, reconnectDelay)
			time.Sleep(reconnectDelay)
			continue
		}
	}
}

func (c *Client) connect() error {
	u, err := url.Parse(c.cfg.ServerURL + "/ws/pc?token=" + c.cfg.PCSecret)
	if err != nil {
		return err
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	c.conn = conn

	log.Println("Connected to relay server")

	// Запускаем ffmpeg
	if err := c.enc.Start(); err != nil {
		return err
	}
	defer c.enc.Stop()

	// Горутина: читаем input события от браузера
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			c.input.Handle(msg)
		}
	}()

	// Основной поток: читаем H.264 из ffmpeg → отправляем на сервер
	buf := make([]byte, chunkSize)
	for {
		n, err := c.enc.Stdout.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
			return err
		}
	}
}

func (c *Client) Stop() {
	close(c.done)
	if c.conn != nil {
		c.conn.Close()
	}
}

