package mac

//type ActivationPolicy int
//
//const (
//	NSApplicationActivationPolicyRegular    ActivationPolicy = 0
//	NSApplicationActivationPolicyAccessory  ActivationPolicy = 1
//	NSApplicationActivationPolicyProhibited ActivationPolicy = 2
//)

type AboutInfo struct {
	Title   string
	Message string
	Icon    []byte
}

// Options are options specific to Mac
type Options struct {
	TitleBar             *TitleBar
	Appearance           AppearanceType
	WebviewIsTransparent bool
	WindowIsTranslucent  bool
	// WindowCanJoinAllSpaces makes the main window appear on all Desktops (Spaces).
	// Equivalent to macOS "Assign to: All Desktops" for this window.
	WindowCanJoinAllSpaces bool
	// WindowFullScreenAuxiliary allows the window to appear in fullscreen Spaces as an auxiliary window.
	// Commonly used together with WindowCanJoinAllSpaces.
	WindowFullScreenAuxiliary bool
	Preferences          *Preferences
	DisableZoom          bool
	// ActivationPolicy     ActivationPolicy
	About      *AboutInfo
	OnFileOpen func(filePath string) `json:"-"`
	OnUrlOpen  func(filePath string) `json:"-"`
	// URLHandlers          map[string]func(string)
}
