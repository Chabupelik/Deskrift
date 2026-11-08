package input

import (
	"encoding/json"
	"log"
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	procSendInput  = user32.NewProc("SendInput")
	procSetCursorPos = user32.NewProc("SetCursorPos")
)

const (
	inputMouse    = 0
	inputKeyboard = 1

	mouseeventfMove        = 0x0001
	mouseeventfLeftdown    = 0x0002
	mouseeventfLeftup      = 0x0004
	mouseeventfRightdown   = 0x0008
	mouseeventfRightup     = 0x0010
	mouseeventfAbsolute    = 0x8000

	keyeventfKeyup = 0x0002
)

type mouseInput struct {
	inputType uint32
	dx        int32
	dy        int32
	mouseData uint32
	dwFlags   uint32
	time      uint32
	extraInfo uintptr
	_         [8]byte
}

type keyboardInput struct {
	inputType uint32
	wVk       uint16
	wScan     uint16
	dwFlags   uint32
	time      uint32
	extraInfo uintptr
	_         [8]byte
}

type InputEvent struct {
	Type   string `json:"type"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Button string `json:"button,omitempty"`
	Key    int    `json:"key,omitempty"`
	Down   bool   `json:"down,omitempty"`
}

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Handle(data []byte) {
	var event InputEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("input parse error: %v", err)
		return
	}

	switch event.Type {
	case "mousemove":
		h.mouseMove(event.X, event.Y)
	case "mousedown":
		h.mouseClick(event.Button, true)
	case "mouseup":
		h.mouseClick(event.Button, false)
	case "keydown":
		h.keyEvent(event.Key, true)
	case "keyup":
		h.keyEvent(event.Key, false)
	}
}

func (h *Handler) mouseMove(x, y int) {
	procSetCursorPos.Call(uintptr(x), uintptr(y))
}

func (h *Handler) mouseClick(button string, down bool) {
	var flags uint32
	switch button {
	case "left":
		if down {
			flags = mouseeventfLeftdown
		} else {
			flags = mouseeventfLeftup
		}
	case "right":
		if down {
			flags = mouseeventfRightdown
		} else {
			flags = mouseeventfRightup
		}
	}

	inp := mouseInput{inputType: inputMouse, dwFlags: flags}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

func (h *Handler) keyEvent(vkCode int, down bool) {
	var flags uint32
	if !down {
		flags = keyeventfKeyup
	}

	inp := keyboardInput{
		inputType: inputKeyboard,
		wVk:       uint16(vkCode),
		dwFlags:   flags,
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

