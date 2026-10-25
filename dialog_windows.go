//go:build windows

package main

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Native Windows 10/11 "browse" dialog (IFileOpenDialog).

var (
	ole32            = windows.NewLazySystemDLL("ole32.dll")
	shell32          = windows.NewLazySystemDLL("shell32.dll")
	procCoInit       = ole32.NewProc("CoInitializeEx")
	procCoUninit     = ole32.NewProc("CoUninitialize")
	procCoCreate     = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree = ole32.NewProc("CoTaskMemFree")
	procSHCreateItem = shell32.NewProc("SHCreateItemFromParsingName")
	mainHwnd         uintptr
)

var (
	clsidFileOpenDialog = windows.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidFileOpenDialog   = windows.GUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidShellItem        = windows.GUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
)

type filterSpec struct{ name, spec *uint16 }

func vcall(obj uintptr, idx int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

// browseDialog shows the native picker. kind: "folder" or "demo".
// Returns "" when the user cancels.
func browseDialog(kind, title, start string) (string, error) {
	type result struct {
		path string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		procCoInit.Call(0, 0x2|0x4) // apartment threaded, disable OLE1 DDE
		defer procCoUninit.Call()
		var dlg uintptr
		hr, _, _ := procCoCreate.Call(uintptr(unsafe.Pointer(&clsidFileOpenDialog)), 0, 1|4, uintptr(unsafe.Pointer(&iidFileOpenDialog)), uintptr(unsafe.Pointer(&dlg)))
		if hr != 0 || dlg == 0 {
			ch <- result{"", errors.New("could not open the Windows file dialog")}
			return
		}
		defer vcall(dlg, 2)
		opts := uintptr(0x40 | 0x800) // force file system, path must exist
		if kind == "folder" {
			opts |= 0x20 // pick folders
		} else {
			opts |= 0x1000 // file must exist
			n1, _ := windows.UTF16PtrFromString("CS 1.6 demos (*.dem)")
			s1, _ := windows.UTF16PtrFromString("*.dem")
			n2, _ := windows.UTF16PtrFromString("All files (*.*)")
			s2, _ := windows.UTF16PtrFromString("*.*")
			specs := []filterSpec{{n1, s1}, {n2, s2}}
			vcall(dlg, 4, 2, uintptr(unsafe.Pointer(&specs[0])))
		}
		vcall(dlg, 9, opts)
		if t, err := windows.UTF16PtrFromString(title); err == nil {
			vcall(dlg, 17, uintptr(unsafe.Pointer(t)))
		}
		if start != "" {
			if sp, err := windows.UTF16PtrFromString(start); err == nil {
				var item uintptr
				if r, _, _ := procSHCreateItem.Call(uintptr(unsafe.Pointer(sp)), 0, uintptr(unsafe.Pointer(&iidShellItem)), uintptr(unsafe.Pointer(&item))); r == 0 && item != 0 {
					vcall(dlg, 12, item)
					vcall(item, 2)
				}
			}
		}
		if hr := vcall(dlg, 3, mainHwnd); hr != 0 {
			ch <- result{"", nil} // cancelled
			return
		}
		var item uintptr
		if vcall(dlg, 20, uintptr(unsafe.Pointer(&item))) != 0 || item == 0 {
			ch <- result{"", nil}
			return
		}
		defer vcall(item, 2)
		var ps *uint16
		if vcall(item, 5, 0x80058000, uintptr(unsafe.Pointer(&ps))) != 0 || ps == nil {
			ch <- result{"", errors.New("could not read the chosen path")}
			return
		}
		path := windows.UTF16PtrToString(ps)
		procCoTaskMemFree.Call(uintptr(unsafe.Pointer(ps)))
		ch <- result{path, nil}
	}()
	r := <-ch
	return r.path, r.err
}
