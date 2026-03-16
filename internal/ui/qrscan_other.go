//go:build !android

package ui

// openCameraForQR is a no-op on non-Android platforms.
func openCameraForQR() bool { return false }

// readCapturedPhoto is a no-op on non-Android platforms.
func readCapturedPhoto() []byte { return nil }

// cleanupPhotoURI is a no-op on non-Android platforms.
func cleanupPhotoURI() {}
