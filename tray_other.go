//go:build !windows

package main

import "github.com/wailsapp/wails/v2/pkg/runtime"

// startTray is a stub for macOS and Linux.
// System tray is only supported on Windows in this application.
func (a *App) startTray() {
	// No tray support on non-Windows platforms.
}

// ShowWindow restores and focuses the application window on non-Windows platforms.
func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

// QuitApp properly quits the application on non-Windows platforms.
func (a *App) QuitApp() {
	a.quitting = true
	runtime.Quit(a.ctx)
}
