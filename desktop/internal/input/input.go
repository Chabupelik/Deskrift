package input

import (
	"encoding/json"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procSendInput        = user32.NewProc("SendInput")
	procSetCursorPos     = user32.NewProc("SetCursorPos")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
)

const (
	inputMouse    = 0
	inputKeyboard = 1

	mouseeventfMove       = 0x0001
	mouseeventfLeftdown   = 0x0002
	mouseeventfLeftup     = 0x0004
	mouseeventfRightdown  = 0x0008
	mouseeventfRightup    = 0x0010
	mouseeventfMiddledown = 0x0020
	mouseeventfMiddleup   = 0x0040
	mouseeventfWheel      = 0x0800

	keyeventfKeyup = 0x0002

	vkControl = 0x11
	vkV       = 0x56
)

type mouseInput struct {
	dx        int32
	dy        int32
	mouseData uint32
	dwFlags   uint32
	time      uint32
	_         uint32
	extraInfo uintptr
}

type keybdInput struct {
	wVk       uint16
	wScan     uint16
	dwFlags   uint32
	time      uint32
	_         uint32
	extraInfo uintptr
	_         [8]byte
}

type winInput struct {
	inputType uint32
	_         uint32
	data      [32]byte
}

type InputEvent struct {
	Type   string `json:"type"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Button string `json:"button,omitempty"`
	Key    int    `json:"key,omitempty"`
	Delta  int    `json:"delta,omitempty"`
	Text   string `json:"text,omitempty"`
}

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Handle(data []byte) {
	var event InputEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return
	}

	switch event.Type {
	case "mousemove":
		h.mouseMove(event.X, event.Y)
	case "mousedown":
		h.mouseClick(event.Button, true)
	case "mouseup":
		h.mouseClick(event.Button, false)
	case "wheel":
		h.mouseWheel(event.Delta)
	case "keydown":
		h.keyEvent(event.Key, true)
	case "keyup":
		h.keyEvent(event.Key, false)
	case "clipboard":
		if event.Text != "" {
			h.pasteText(event.Text)
		}
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
	case "middle":
		if down {
			flags = mouseeventfMiddledown
		} else {
			flags = mouseeventfMiddleup
		}
	default:
		return
	}

	mi := mouseInput{dwFlags: flags}
	var inp winInput
	inp.inputType = inputMouse
	*(*mouseInput)(unsafe.Pointer(&inp.data[0])) = mi
	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

func (h *Handler) mouseWheel(delta int) {
	mi := mouseInput{
		dwFlags:   mouseeventfWheel,
		mouseData: uint32(int32(delta)),
	}
	var inp winInput
	inp.inputType = inputMouse
	*(*mouseInput)(unsafe.Pointer(&inp.data[0])) = mi
	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

func (h *Handler) keyEvent(vkCode int, down bool) {
	var flags uint32
	if !down {
		flags = keyeventfKeyup
	}

	ki := keybdInput{
		wVk:     uint16(vkCode),
		dwFlags: flags,
	}
	var inp winInput
	inp.inputType = inputKeyboard
	*(*keybdInput)(unsafe.Pointer(&inp.data[0])) = ki
	procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
}

func (h *Handler) pasteText(text string) {
	u16, err := syscall.UTF16FromString(text)
	if err != nil {
		return
	}
	bytesCount := len(u16) * 2

	hMem, _, _ := procGlobalAlloc.Call(0x0002, uintptr(bytesCount))
	if hMem == 0 {
		return
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return
	}

	dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), bytesCount)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&u16[0])), bytesCount)
	copy(dst, src)
	procGlobalUnlock.Call(hMem)

	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()
	procSetClipboardData.Call(13, hMem)

	time.Sleep(10 * time.Millisecond)
	h.keyEvent(vkControl, true)
	h.keyEvent(vkV, true)
	h.keyEvent(vkV, false)
	h.keyEvent(vkControl, false)
}
