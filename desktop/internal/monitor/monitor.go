package monitor

import (
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfo      = user32.NewProc("GetMonitorInfoW")
)

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type monitorInfoEx struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
	SzDevice  [32]uint16
}

type Monitor struct {
	Index   int
	Left    int
	Top     int
	Right   int
	Bottom  int
	Width   int
	Height  int
	Primary bool
}

func GetAllMonitors() []Monitor {
	var monitors []Monitor
	idx := 0
	cb := syscall.NewCallback(func(hMonitor, hdcMonitor, lprcMonitor, dwData uintptr) uintptr {
		var mi monitorInfoEx
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		ret, _, _ := procGetMonitorInfo.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
		if ret != 0 {
			left := int(mi.RcMonitor.Left)
			top := int(mi.RcMonitor.Top)
			right := int(mi.RcMonitor.Right)
			bottom := int(mi.RcMonitor.Bottom)
			isPrimary := (mi.DwFlags & 1) != 0
			monitors = append(monitors, Monitor{
				Index:   idx,
				Left:    left,
				Top:     top,
				Right:   right,
				Bottom:  bottom,
				Width:   right - left,
				Height:  bottom - top,
				Primary: isPrimary,
			})
			idx++
		}
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, cb, 0)
	if len(monitors) == 0 {
		monitors = append(monitors, Monitor{
			Index:   0,
			Left:    0,
			Top:     0,
			Right:   1920,
			Bottom:  1080,
			Width:   1920,
			Height:  1080,
			Primary: true,
		})
	}
	return monitors
}
