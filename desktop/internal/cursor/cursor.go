package cursor

import (
	"syscall"
	"unsafe"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procGetCursorInfo = user32.NewProc("GetCursorInfo")
	procLoadCursorW   = user32.NewProc("LoadCursorW")
)

// Standard cursor IDs
const (
	idcArrow      = 32512
	idcIBeam      = 32513
	idcWait       = 32514
	idcCross      = 32515
	idcSizeNWSE   = 32642
	idcSizeNESW   = 32643
	idcSizeWE     = 32644
	idcSizeNS     = 32645
	idcSizeAll    = 32646
	idcNo         = 32648
	idcHand       = 32649
	idcAppStarting = 32650
)

type cursorInfo struct {
	cbSize uint32
	flags  uint32
	hCursor uintptr
	ptX    int32
	ptY    int32
}

var stdCursors map[uintptr]string

func init() {
	stdCursors = make(map[uintptr]string)
	pairs := []struct {
		id  uintptr
		css string
	}{
		{idcArrow, "default"},
		{idcIBeam, "text"},
		{idcWait, "wait"},
		{idcCross, "crosshair"},
		{idcSizeNWSE, "nwse-resize"},
		{idcSizeNESW, "nesw-resize"},
		{idcSizeWE, "ew-resize"},
		{idcSizeNS, "ns-resize"},
		{idcSizeAll, "move"},
		{idcNo, "not-allowed"},
		{idcHand, "pointer"},
		{idcAppStarting, "progress"},
	}
	for _, p := range pairs {
		h, _, _ := procLoadCursorW.Call(0, p.id)
		if h != 0 {
			stdCursors[h] = p.css
		}
	}
}

// GetCSS returns the CSS cursor name for the current system cursor.
func GetCSS() string {
	var info cursorInfo
	info.cbSize = uint32(unsafe.Sizeof(info))
	procGetCursorInfo.Call(uintptr(unsafe.Pointer(&info)))

	if css, ok := stdCursors[info.hCursor]; ok {
		return css
	}
	// Unknown cursor — treat as default
	return "default"
}
