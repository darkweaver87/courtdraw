//go:build ios

package ui

// systemBeep is a no-op on mobile platforms.
// TODO: implement native audio via NDK/OpenSLES (Android) or AVAudioPlayer (iOS).
func systemBeep() {}
