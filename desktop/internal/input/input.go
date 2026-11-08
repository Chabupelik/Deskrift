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

	procSetCursorPos     = user32.NewProc("SetCursorPos")
	procMouseEvent       = user32.NewProc("mouse_event")
	procKeybdEvent       = user32.NewProc("keybd_event")
	procMapVirtualKey    = user32.NewProc("MapVirtualKeyW")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
)

const (
	mouseeventfLeftdown   = 0x0002
	mouseeventfLeftup     = 0x0004
	mouseeventfRightdown  = 0x0008
	mouseeventfRightup    = 0x0010
	mouseeventfMiddledown = 0x0020
	mouseeventfMiddleup   = 0x0040
	mouseeventfWheel      = 0x0800

	keyeventfExtendedkey = 0x0001
	keyeventfKeyup       = 0x0002

	vkControl = 0x11
	vkV       = 0x56
)

type InputEvent struct {
	Type   string `json:"type"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Button string `json:"button,omitempty"`
	Key    int    `json:"key,omitempty"`
	Delta  int    `json:"delta,omitempty"`
	Text   string `json:"text,omitempty"`
}

type Handler struct {
	lastX int
	lastY int
}

func New() *Handler {
	return &Handler{
		lastX: -1,
		lastY: -1,
	}
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
	if h.lastX == x && h.lastY == y {
		return
	}
	h.lastX = x
	h.lastY = y
	procSetCursorPos.Call(uintptr(x), uintptr(y))
}

func (h *Handler) mouseClick(button string, down bool) {
	var flags uintptr
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
	procMouseEvent.Call(flags, 0, 0, 0, 0)
}

func (h *Handler) mouseWheel(delta int) {
	procMouseEvent.Call(mouseeventfWheel, 0, 0, uintptr(int32(delta)), 0)
}

func (h *Handler) keyEvent(vkCode int, down bool) {
	var flags uintptr
	if vkCode >= 0x21 && vkCode <= 0x28 {
		flags |= keyeventfExtendedkey
	}
	if !down {
		flags |= keyeventfKeyup
	}
	scanCode, _, _ := procMapVirtualKey.Call(uintptr(vkCode), 0)
	procKeybdEvent.Call(uintptr(vkCode), scanCode, flags, 0)
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
