//go:build windows

package windows

import (
	"time"
	"unsafe"

	"github.com/wailsapp/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
)

// Hidden-window power management for WebView2, enabled by
// windows.Options.WebviewLowMemoryWhenHidden.
//
// Hiding the window only hides the HWND; the WebView2 controller still thinks
// it is visible and keeps running at full tilt. While the window is hidden we
// mark the controller invisible and lower its memory usage target, which lets
// WebView2 trim caches and working set, and once the page has been idle for a
// moment we suspend it, which pauses script timers and releases more. Anything
// that needs the page (showing the window, running script, navigating)
// resumes it first. Every function here runs on the UI thread.

// Vtable slots, from the method counts in WebView2.idl: IUnknown (3),
// ICoreWebView2 (58), ICoreWebView2_2 (7) and ICoreWebView2_3 to _18 (51).
const (
	slotTrySuspend                = 3 + 58 + 7
	slotResume                    = slotTrySuspend + 1
	slotPutMemoryUsageTargetLevel = 3 + 58 + 7 + 51 + 1
)

const (
	memoryUsageTargetLevelNormal = 0
	memoryUsageTargetLevelLow    = 1
)

// webviewSuspendDelay keeps a quick hide/show from bouncing through a
// suspend, and re-arms the suspend after script woke a hidden page.
const webviewSuspendDelay = 10 * time.Second

var iidICoreWebView2_19 = edge.NewGUID("{6921f954-79b0-437f-a997-c85811897c68}")

// comCall invokes the method at the given vtable slot of a COM object.
//
//go:uintptrescapes
func comCall(obj unsafe.Pointer, slot int, args ...uintptr) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	fn := *(*edge.ComProc)(unsafe.Add(vtbl, slot*int(unsafe.Sizeof(uintptr(0)))))
	hr, _, _ := fn.Call(append([]uintptr{uintptr(obj)}, args...)...)
	return hr
}

// trySuspendCompletedHandler is a static ICoreWebView2TrySuspendCompletedHandler.
// The result is not needed: a failed suspend just leaves the page running.
type trySuspendCompletedHandler struct {
	vtbl *trySuspendCompletedHandlerVtbl
}

type trySuspendCompletedHandlerVtbl struct {
	QueryInterface edge.ComProc
	AddRef         edge.ComProc
	Release        edge.ComProc
	Invoke         edge.ComProc
}

var trySuspendCompleted = &trySuspendCompletedHandler{
	vtbl: &trySuspendCompletedHandlerVtbl{
		QueryInterface: edge.NewComProc(func(this *trySuspendCompletedHandler, _ uintptr, object *uintptr) uintptr {
			*object = uintptr(unsafe.Pointer(this))
			return uintptr(windows.S_OK)
		}),
		AddRef:  edge.NewComProc(func(*trySuspendCompletedHandler) uintptr { return 1 }),
		Release: edge.NewComProc(func(*trySuspendCompletedHandler) uintptr { return 1 }),
		Invoke: edge.NewComProc(func(*trySuspendCompletedHandler, uintptr, uintptr) uintptr {
			return uintptr(windows.S_OK)
		}),
	},
}

// webviewLowMemoryEnabled also guards against a webview whose controller is
// not created yet: every call below dereferences it.
func (f *Frontend) webviewLowMemoryEnabled() bool {
	return f.frontendOptions.Windows != nil && f.frontendOptions.Windows.WebviewLowMemoryWhenHidden &&
		f.chromium != nil && f.chromium.GetController() != nil
}

// webviewEnterLowMemory is called after the window was hidden.
func (f *Frontend) webviewEnterLowMemory() {
	if !f.webviewLowMemoryEnabled() || f.webviewHidden {
		return
	}
	f.webviewHidden = true
	if err := f.chromium.Hide(); err != nil {
		f.logger.Warning("webview hide failed: %v", err)
	}
	f.setWebviewMemoryUsageTargetLevel(memoryUsageTargetLevelLow)
	f.scheduleWebviewSuspend()
}

// webviewLeaveLowMemory is called before the window is shown.
func (f *Frontend) webviewLeaveLowMemory() {
	if !f.webviewLowMemoryEnabled() || !f.webviewHidden {
		return
	}
	f.webviewHidden = false
	f.webviewSuspendGen++ // cancel a pending suspend
	f.resumeWebview()
	f.setWebviewMemoryUsageTargetLevel(memoryUsageTargetLevelNormal)
	if err := f.chromium.Show(); err != nil {
		f.logger.Warning("webview show failed: %v", err)
	}
}

// wakeWebview lets script run in a hidden, suspended page; the page stays in
// low-memory mode and is suspended again once idle.
func (f *Frontend) wakeWebview() {
	if !f.webviewSuspended {
		return
	}
	f.resumeWebview()
	if f.webviewHidden {
		f.scheduleWebviewSuspend()
	}
}

func (f *Frontend) scheduleWebviewSuspend() {
	f.webviewSuspendGen++
	gen := f.webviewSuspendGen
	time.AfterFunc(webviewSuspendDelay, func() {
		f.mainWindow.Invoke(func() {
			if f.webviewHidden && !f.webviewSuspended && f.webviewSuspendGen == gen {
				f.suspendWebview()
			}
		})
	})
}

func (f *Frontend) suspendWebview() {
	webview := f.chromium.GetICoreWebView2_3()
	if webview == nil {
		return
	}
	defer webview.Release()
	if hr := comCall(unsafe.Pointer(webview), slotTrySuspend, uintptr(unsafe.Pointer(trySuspendCompleted))); windows.Handle(hr) != windows.S_OK {
		f.logger.Warning("webview TrySuspend failed: %v", windows.Errno(hr))
		return
	}
	f.webviewSuspended = true
}

func (f *Frontend) resumeWebview() {
	if !f.webviewSuspended {
		return
	}
	f.webviewSuspended = false
	webview := f.chromium.GetICoreWebView2_3()
	if webview == nil {
		return
	}
	defer webview.Release()
	if hr := comCall(unsafe.Pointer(webview), slotResume); windows.Handle(hr) != windows.S_OK {
		f.logger.Warning("webview Resume failed: %v", windows.Errno(hr))
	}
}

// setWebviewMemoryUsageTargetLevel is a no-op on WebView2 runtimes older than
// ICoreWebView2_19 (1.0.2210).
func (f *Frontend) setWebviewMemoryUsageTargetLevel(level uintptr) {
	webview := f.chromium.GetICoreWebView2_3()
	if webview == nil {
		return
	}
	defer webview.Release()
	var webview19 unsafe.Pointer
	if hr := comCall(unsafe.Pointer(webview), 0, uintptr(unsafe.Pointer(iidICoreWebView2_19)), uintptr(unsafe.Pointer(&webview19))); windows.Handle(hr) != windows.S_OK || webview19 == nil {
		return
	}
	defer comCall(webview19, 2) // Release
	if hr := comCall(webview19, slotPutMemoryUsageTargetLevel, level); windows.Handle(hr) != windows.S_OK {
		f.logger.Warning("webview put_MemoryUsageTargetLevel failed: %v", windows.Errno(hr))
	}
}
