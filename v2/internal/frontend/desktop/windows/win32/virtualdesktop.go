//go:build windows

package win32

import (
	"syscall"
	"unsafe"
)

// GUID represents a Windows GUID structure
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	modole32             = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx   = modole32.NewProc("CoInitializeEx")
	procCoUninitialize   = modole32.NewProc("CoUninitialize")
	procCoCreateInstance = modole32.NewProc("CoCreateInstance")
)

// COM initialization constants
const (
	COINIT_APARTMENTTHREADED = 0x2
	COINIT_MULTITHREADED     = 0x0
	CLSCTX_LOCAL_SERVER      = 0x4
)

// GUIDs for Virtual Desktop API
var (
	// ImmersiveShell CLSID - same across Windows 10/11
	CLSID_ImmersiveShell = GUID{0xC2F03A33, 0x21F5, 0x47FA, [8]byte{0xB4, 0xBB, 0x15, 0x63, 0x62, 0xA2, 0xF2, 0x39}}

	// IServiceProvider IID
	IID_IServiceProvider = GUID{0x6D5140C1, 0x7436, 0x11CE, [8]byte{0x80, 0x34, 0x00, 0xAA, 0x00, 0x60, 0x09, 0xFA}}

	// IApplicationViewCollection CLSID and IID
	CLSID_IApplicationViewCollection = GUID{0x1841C6D7, 0x4F9D, 0x42C0, [8]byte{0xAF, 0x41, 0x87, 0x47, 0x21, 0x26, 0x74, 0xBA}}

	// Windows 11 24H2+ (Build 26100+)
	IID_IVirtualDesktopPinnedApps_Win11_24H2 = GUID{0x92754B94, 0x0C45, 0x4D16, [8]byte{0x9F, 0x52, 0x55, 0xD4, 0xD5, 0x10, 0x40, 0xCE}}

	// Windows 11 22H2-23H2 (Build 22621-26099)
	IID_IVirtualDesktopPinnedApps_Win11_22H2 = GUID{0x4CE81583, 0x1E4C, 0x4632, [8]byte{0xA6, 0x21, 0x07, 0xA5, 0x35, 0x43, 0x14, 0x8F}}

	// Windows 11 21H2 (Build 22000)
	IID_IVirtualDesktopPinnedApps_Win11_21H2 = GUID{0x4CE81583, 0x1E4C, 0x4632, [8]byte{0xA6, 0x21, 0x07, 0xA5, 0x35, 0x43, 0x14, 0x8F}}

	// Windows 10 (Build 17134+)
	IID_IVirtualDesktopPinnedApps_Win10 = GUID{0x4CE81583, 0x1E4C, 0x4632, [8]byte{0xA6, 0x21, 0x07, 0xA5, 0x35, 0x43, 0x14, 0x8F}}

	// CLSID for VirtualDesktopPinnedApps service
	CLSID_VirtualDesktopPinnedApps = GUID{0xB5A399E7, 0x1C87, 0x46B8, [8]byte{0x88, 0xE9, 0xFC, 0x57, 0x47, 0xB1, 0x71, 0xBD}}

	// IApplicationView IID for getting view from HWND
	IID_IApplicationViewCollection_Win11 = GUID{0x1841C6D7, 0x4F9D, 0x42C0, [8]byte{0xAF, 0x41, 0x87, 0x47, 0x21, 0x26, 0x74, 0xBA}}
	IID_IApplicationViewCollection_Win10 = GUID{0x2C08ADF0, 0xA386, 0x4B35, [8]byte{0x92, 0x50, 0x04, 0x67, 0x9D, 0x96, 0x19, 0x2F}}
)

// COM interface vtable structures
type IUnknownVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

type IServiceProviderVtbl struct {
	IUnknownVtbl
	QueryService uintptr
}

type IApplicationViewCollectionVtbl struct {
	IUnknownVtbl
	GetViews                       uintptr
	GetViewsByZOrder               uintptr
	GetViewsByAppUserModelId       uintptr
	GetViewForHwnd                 uintptr
	GetViewForApplication          uintptr
	GetViewForAppUserModelId       uintptr
	GetViewInFocus                 uintptr
	Unknown1                       uintptr
	RefreshCollection              uintptr
	RegisterForApplicationViewChanges uintptr
	UnregisterForApplicationViewChanges uintptr
}

type IVirtualDesktopPinnedAppsVtbl struct {
	IUnknownVtbl
	IsAppIdPinned uintptr
	PinAppID      uintptr
	UnpinAppID    uintptr
	IsViewPinned  uintptr
	PinView       uintptr
	UnpinView     uintptr
}

type IServiceProvider struct {
	vtbl *IServiceProviderVtbl
}

type IApplicationViewCollection struct {
	vtbl *IApplicationViewCollectionVtbl
}

type IVirtualDesktopPinnedApps struct {
	vtbl *IVirtualDesktopPinnedAppsVtbl
}

type IApplicationView struct {
	vtbl *IUnknownVtbl // We only need basic interface
}

// IsVirtualDesktopSupported checks if Virtual Desktop API is available
func IsVirtualDesktopSupported() bool {
	// Virtual Desktop pinning requires Windows 10 1803 (Build 17134) or later
	return IsWindowsVersionAtLeast(10, 0, 17134)
}

// getVirtualDesktopPinnedAppsIID returns the correct IID based on Windows version
func getVirtualDesktopPinnedAppsIID() *GUID {
	if windowsVersion.Build >= 26100 {
		// Windows 11 24H2+
		return &IID_IVirtualDesktopPinnedApps_Win11_24H2
	} else if windowsVersion.Build >= 22000 {
		// Windows 11
		return &IID_IVirtualDesktopPinnedApps_Win11_22H2
	}
	// Windows 10
	return &IID_IVirtualDesktopPinnedApps_Win10
}

// getApplicationViewCollectionIID returns the correct IID based on Windows version
func getApplicationViewCollectionIID() *GUID {
	if windowsVersion.Build >= 22000 {
		return &IID_IApplicationViewCollection_Win11
	}
	return &IID_IApplicationViewCollection_Win10
}

// PinWindow pins the specified window to all virtual desktops
// Returns true on success, false on failure (fails silently)
func PinWindow(hwnd uintptr) bool {
	if !IsVirtualDesktopSupported() {
		return false
	}

	// Initialize COM
	hr, _, _ := procCoInitializeEx.Call(0, COINIT_MULTITHREADED)
	if hr != 0 && hr != 1 { // S_OK or S_FALSE (already initialized)
		return false
	}
	defer procCoUninitialize.Call()

	// Create ImmersiveShell instance to get IServiceProvider
	var serviceProvider *IServiceProvider
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&CLSID_ImmersiveShell)),
		0,
		CLSCTX_LOCAL_SERVER,
		uintptr(unsafe.Pointer(&IID_IServiceProvider)),
		uintptr(unsafe.Pointer(&serviceProvider)),
	)
	if hr != 0 || serviceProvider == nil {
		return false
	}
	defer syscall.SyscallN(serviceProvider.vtbl.Release, uintptr(unsafe.Pointer(serviceProvider)))

	// Query for IApplicationViewCollection service
	var viewCollection *IApplicationViewCollection
	appViewCollectionIID := getApplicationViewCollectionIID()
	hr, _, _ = syscall.SyscallN(
		serviceProvider.vtbl.QueryService,
		uintptr(unsafe.Pointer(serviceProvider)),
		uintptr(unsafe.Pointer(&CLSID_IApplicationViewCollection)),
		uintptr(unsafe.Pointer(appViewCollectionIID)),
		uintptr(unsafe.Pointer(&viewCollection)),
	)
	if hr != 0 || viewCollection == nil {
		return false
	}
	defer syscall.SyscallN(viewCollection.vtbl.Release, uintptr(unsafe.Pointer(viewCollection)))

	// Query for IVirtualDesktopPinnedApps service
	var pinnedApps *IVirtualDesktopPinnedApps
	pinnedAppsIID := getVirtualDesktopPinnedAppsIID()
	hr, _, _ = syscall.SyscallN(
		serviceProvider.vtbl.QueryService,
		uintptr(unsafe.Pointer(serviceProvider)),
		uintptr(unsafe.Pointer(&CLSID_VirtualDesktopPinnedApps)),
		uintptr(unsafe.Pointer(pinnedAppsIID)),
		uintptr(unsafe.Pointer(&pinnedApps)),
	)
	if hr != 0 || pinnedApps == nil {
		return false
	}
	defer syscall.SyscallN(pinnedApps.vtbl.Release, uintptr(unsafe.Pointer(pinnedApps)))

	// Get IApplicationView for the window
	var view *IApplicationView
	hr, _, _ = syscall.SyscallN(
		viewCollection.vtbl.GetViewForHwnd,
		uintptr(unsafe.Pointer(viewCollection)),
		hwnd,
		uintptr(unsafe.Pointer(&view)),
	)
	if hr != 0 || view == nil {
		return false
	}
	defer syscall.SyscallN(view.vtbl.Release, uintptr(unsafe.Pointer(view)))

	// Pin the view to all virtual desktops
	hr, _, _ = syscall.SyscallN(
		pinnedApps.vtbl.PinView,
		uintptr(unsafe.Pointer(pinnedApps)),
		uintptr(unsafe.Pointer(view)),
	)

	return hr == 0
}

// UnpinWindow unpins the specified window from all virtual desktops
// Returns true on success, false on failure (fails silently)
func UnpinWindow(hwnd uintptr) bool {
	if !IsVirtualDesktopSupported() {
		return false
	}

	// Initialize COM
	hr, _, _ := procCoInitializeEx.Call(0, COINIT_MULTITHREADED)
	if hr != 0 && hr != 1 {
		return false
	}
	defer procCoUninitialize.Call()

	// Create ImmersiveShell instance
	var serviceProvider *IServiceProvider
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&CLSID_ImmersiveShell)),
		0,
		CLSCTX_LOCAL_SERVER,
		uintptr(unsafe.Pointer(&IID_IServiceProvider)),
		uintptr(unsafe.Pointer(&serviceProvider)),
	)
	if hr != 0 || serviceProvider == nil {
		return false
	}
	defer syscall.SyscallN(serviceProvider.vtbl.Release, uintptr(unsafe.Pointer(serviceProvider)))

	// Query for IApplicationViewCollection
	var viewCollection *IApplicationViewCollection
	appViewCollectionIID := getApplicationViewCollectionIID()
	hr, _, _ = syscall.SyscallN(
		serviceProvider.vtbl.QueryService,
		uintptr(unsafe.Pointer(serviceProvider)),
		uintptr(unsafe.Pointer(&CLSID_IApplicationViewCollection)),
		uintptr(unsafe.Pointer(appViewCollectionIID)),
		uintptr(unsafe.Pointer(&viewCollection)),
	)
	if hr != 0 || viewCollection == nil {
		return false
	}
	defer syscall.SyscallN(viewCollection.vtbl.Release, uintptr(unsafe.Pointer(viewCollection)))

	// Query for IVirtualDesktopPinnedApps
	var pinnedApps *IVirtualDesktopPinnedApps
	pinnedAppsIID := getVirtualDesktopPinnedAppsIID()
	hr, _, _ = syscall.SyscallN(
		serviceProvider.vtbl.QueryService,
		uintptr(unsafe.Pointer(serviceProvider)),
		uintptr(unsafe.Pointer(&CLSID_VirtualDesktopPinnedApps)),
		uintptr(unsafe.Pointer(pinnedAppsIID)),
		uintptr(unsafe.Pointer(&pinnedApps)),
	)
	if hr != 0 || pinnedApps == nil {
		return false
	}
	defer syscall.SyscallN(pinnedApps.vtbl.Release, uintptr(unsafe.Pointer(pinnedApps)))

	// Get IApplicationView for the window
	var view *IApplicationView
	hr, _, _ = syscall.SyscallN(
		viewCollection.vtbl.GetViewForHwnd,
		uintptr(unsafe.Pointer(viewCollection)),
		hwnd,
		uintptr(unsafe.Pointer(&view)),
	)
	if hr != 0 || view == nil {
		return false
	}
	defer syscall.SyscallN(view.vtbl.Release, uintptr(unsafe.Pointer(view)))

	// Unpin the view
	hr, _, _ = syscall.SyscallN(
		pinnedApps.vtbl.UnpinView,
		uintptr(unsafe.Pointer(pinnedApps)),
		uintptr(unsafe.Pointer(view)),
	)

	return hr == 0
}

// IsWindowPinned checks if the window is pinned to all virtual desktops
func IsWindowPinned(hwnd uintptr) bool {
	if !IsVirtualDesktopSupported() {
		return false
	}

	// Initialize COM
	hr, _, _ := procCoInitializeEx.Call(0, COINIT_MULTITHREADED)
	if hr != 0 && hr != 1 {
		return false
	}
	defer procCoUninitialize.Call()

	// Create ImmersiveShell instance
	var serviceProvider *IServiceProvider
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&CLSID_ImmersiveShell)),
		0,
		CLSCTX_LOCAL_SERVER,
		uintptr(unsafe.Pointer(&IID_IServiceProvider)),
		uintptr(unsafe.Pointer(&serviceProvider)),
	)
	if hr != 0 || serviceProvider == nil {
		return false
	}
	defer syscall.SyscallN(serviceProvider.vtbl.Release, uintptr(unsafe.Pointer(serviceProvider)))

	// Query for IApplicationViewCollection
	var viewCollection *IApplicationViewCollection
	appViewCollectionIID := getApplicationViewCollectionIID()
	hr, _, _ = syscall.SyscallN(
		serviceProvider.vtbl.QueryService,
		uintptr(unsafe.Pointer(serviceProvider)),
		uintptr(unsafe.Pointer(&CLSID_IApplicationViewCollection)),
		uintptr(unsafe.Pointer(appViewCollectionIID)),
		uintptr(unsafe.Pointer(&viewCollection)),
	)
	if hr != 0 || viewCollection == nil {
		return false
	}
	defer syscall.SyscallN(viewCollection.vtbl.Release, uintptr(unsafe.Pointer(viewCollection)))

	// Query for IVirtualDesktopPinnedApps
	var pinnedApps *IVirtualDesktopPinnedApps
	pinnedAppsIID := getVirtualDesktopPinnedAppsIID()
	hr, _, _ = syscall.SyscallN(
		serviceProvider.vtbl.QueryService,
		uintptr(unsafe.Pointer(serviceProvider)),
		uintptr(unsafe.Pointer(&CLSID_VirtualDesktopPinnedApps)),
		uintptr(unsafe.Pointer(pinnedAppsIID)),
		uintptr(unsafe.Pointer(&pinnedApps)),
	)
	if hr != 0 || pinnedApps == nil {
		return false
	}
	defer syscall.SyscallN(pinnedApps.vtbl.Release, uintptr(unsafe.Pointer(pinnedApps)))

	// Get IApplicationView for the window
	var view *IApplicationView
	hr, _, _ = syscall.SyscallN(
		viewCollection.vtbl.GetViewForHwnd,
		uintptr(unsafe.Pointer(viewCollection)),
		hwnd,
		uintptr(unsafe.Pointer(&view)),
	)
	if hr != 0 || view == nil {
		return false
	}
	defer syscall.SyscallN(view.vtbl.Release, uintptr(unsafe.Pointer(view)))

	// Check if pinned
	var isPinned int32
	hr, _, _ = syscall.SyscallN(
		pinnedApps.vtbl.IsViewPinned,
		uintptr(unsafe.Pointer(pinnedApps)),
		uintptr(unsafe.Pointer(view)),
		uintptr(unsafe.Pointer(&isPinned)),
	)

	return hr == 0 && isPinned != 0
}
