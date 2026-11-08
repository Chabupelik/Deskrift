package relay

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"deskrift/internal/config"
	"deskrift/internal/encoder"
	"deskrift/internal/input"
)

const (
	chunkSize      = 32 * 1024
	reconnectDelay = 3 * time.Second
)

const (
	StatusNotConfigured = "Требуется настройка"
	StatusDisconnected  = "Отключено"
	StatusConnecting    = "Подключение..."
	StatusStandby       = "В сети (ожидание)"
	StatusStreaming     = "Трансляция активна"
)

type ControlMessage struct {
	Type string `json:"type"`
}

type Client struct {
	mu             sync.Mutex
	cfg            *config.Config
	enc            *encoder.Encoder
	input          *input.Handler
	conn           *websocket.Conn
	done           chan struct{}
	streamCancel   chan struct{}
	status         string
	onStatusChange func(string)
}

func New(cfg *config.Config, enc *encoder.Encoder, inp *input.Handler) *Client {
	initialStatus := StatusDisconnected
	if cfg.ServerURL == "" || cfg.PCSecret == "" {
		initialStatus = StatusNotConfigured
	}

	return &Client{
		cfg:          cfg,
		enc:          enc,
		input:        inp,
		done:         make(chan struct{}),
		status:       initialStatus,
		streamCancel: make(chan struct{}),
	}
}

func (c *Client) SetStatusCallback(cb func(string)) {
	c.mu.Lock()
	c.onStatusChange = cb
	current := c.status
	c.mu.Unlock()

	if cb != nil {
		cb(current)
	}
}

func (c *Client) setStatus(s string) {
	c.mu.Lock()
	c.status = s
	cb := c.onStatusChange
	c.mu.Unlock()

	if cb != nil {
		cb(s)
	}
}

func (c *Client) GetStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

func (c *Client) UpdateConfig(newCfg *config.Config) {
	c.mu.Lock()
	c.cfg = newCfg
	conn := c.conn
	c.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
}

func (c *Client) Run() error {
	for {
		select {
		case <-c.done:
			return nil
		default:
		}

		c.mu.Lock()
		serverURL := strings.TrimSpace(c.cfg.ServerURL)
		pcSecret := strings.TrimSpace(c.cfg.PCSecret)
		c.mu.Unlock()

		if serverURL == "" || pcSecret == "" {
			c.setStatus(StatusNotConfigured)
			select {
			case <-c.done:
				return nil
			case <-time.After(2 * time.Second):
				continue
			}
		}

		c.setStatus(StatusConnecting)
		if err := c.connect(serverURL, pcSecret); err != nil {
			c.setStatus(StatusDisconnected)
			select {
			case <-c.done:
				return nil
			case <-time.After(reconnectDelay):
				continue
			}
		}
	}
}

func (c *Client) connect(serverURL, pcSecret string) error {
	targetURL := strings.TrimRight(serverURL, "/") + "/ws/pc?token=" + url.QueryEscape(pcSecret)
	u, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.streamCancel = make(chan struct{})
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.mu.Unlock()
		_ = conn.Close()
		c.stopStreaming()
	}()

	c.setStatus(StatusStandby)

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		if msgType == websocket.TextMessage {
			var ctrl ControlMessage
			if err := json.Unmarshal(msg, &ctrl); err == nil {
				switch ctrl.Type {
				case "start_stream":
					c.startStreaming(conn)
					continue
				case "stop_stream":
					c.stopStreaming()
					continue
				}
			}
			c.input.Handle(msg)
		}
	}
}

func (c *Client) startStreaming(conn *websocket.Conn) {
	c.mu.Lock()
	quality := c.cfg.Quality
	c.mu.Unlock()

	if c.enc.IsRunning() {
		return
	}

	if err := c.enc.Start(quality); err != nil {
		return
	}

	c.mu.Lock()
	c.streamCancel = make(chan struct{})
	cancel := c.streamCancel
	c.mu.Unlock()

	c.setStatus(StatusStreaming)

	go func() {
		defer c.enc.Stop()
		buf := make([]byte, chunkSize)

		for {
			select {
			case <-cancel:
				return
			default:
			}

			if c.enc.Stdout == nil {
				return
			}

			n, err := c.enc.Stdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					fmt.Println(err)
				}
				return
			}

			if n > 0 {
				c.mu.Lock()
				activeConn := c.conn
				c.mu.Unlock()

				if activeConn == nil {
					return
				}

				if err := activeConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
		}
	}()
}

func (c *Client) stopStreaming() {
	c.mu.Lock()
	if c.streamCancel != nil {
		select {
		case <-c.streamCancel:
		default:
			close(c.streamCancel)
		}
	}
	c.mu.Unlock()

	c.enc.Stop()
	if c.GetStatus() == StatusStreaming {
		c.setStatus(StatusStandby)
	}
}

func (c *Client) Stop() {
	close(c.done)
	c.stopStreaming()

	c.mu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.mu.Unlock()
}
