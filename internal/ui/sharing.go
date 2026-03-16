package ui

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/url"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/makiuchi-d/gozxing"
	gozxingqr "github.com/makiuchi-d/gozxing/qrcode"
	qrcode "github.com/skip2/go-qrcode"
	xdraw "golang.org/x/image/draw"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/share"
)

// shareSession creates a bundle from the current session and offers
// two options: share via QR code (cloud upload) or save to file.
func (a *App) shareSession() {
	s := a.sessionTab.Session()
	if s == nil {
		return
	}

	names := share.CollectExerciseNames(s)
	if len(names) == 0 {
		a.statusBar.SetStatus(i18n.T("share.no_exercises"), 1)
		return
	}

	exercises := make(map[string]*model.Exercise)
	for _, name := range names {
		ex, err := a.loadExerciseAny(name)
		if err != nil {
			log.Printf("share: skip exercise %s: %v", name, err)
			continue
		}
		exercises[name] = ex
	}

	dialog.ShowCustomConfirm(
		i18n.T("share.title"),
		i18n.T("share.via_qr"),
		i18n.T("share.save_file"),
		widget.NewLabel(fmt.Sprintf("%d exercises", len(exercises))),
		func(qr bool) {
			if qr {
				a.shareViaQR(s, exercises)
			} else {
				a.shareToFile(s, exercises)
			}
		},
		a.window,
	)
}

func (a *App) shareViaQR(s *model.Session, exercises map[string]*model.Exercise) {
	a.statusBar.SetStatus(i18n.T("share.exporting"), 0)

	go func() {
		var buf bytes.Buffer
		if err := share.CreateBundle(&buf, s, exercises); err != nil {
			fyne.Do(func() {
				a.statusBar.SetStatus(fmt.Sprintf(i18n.T("share.error"), err), 1)
			})
			return
		}

		key, err := share.GenerateKey()
		if err != nil {
			fyne.Do(func() {
				a.statusBar.SetStatus(fmt.Sprintf(i18n.T("share.error"), err), 1)
			})
			return
		}
		encrypted, err := share.Encrypt(key, buf.Bytes())
		if err != nil {
			fyne.Do(func() {
				a.statusBar.SetStatus(fmt.Sprintf(i18n.T("share.error"), err), 1)
			})
			return
		}

		fyne.Do(func() {
			a.statusBar.SetStatus(i18n.T("share.uploading"), 0)
		})
		result, err := share.Upload(context.Background(), encrypted, "session.courtdraw.enc")
		if err != nil {
			fyne.Do(func() {
				dialog.ShowConfirm(
					i18n.T("share.title"),
					i18n.T("share.upload_failed"),
					func(ok bool) {
						if ok {
							a.shareToFile(s, exercises)
						}
					},
					a.window,
				)
			})
			return
		}

		shareURL := result.URL + "#k=" + hex.EncodeToString(key)

		qrImg, err := generateQR(shareURL)
		if err != nil {
			fyne.Do(func() {
				a.statusBar.SetStatus(fmt.Sprintf(i18n.T("share.error"), err), 1)
			})
			return
		}

		fyne.Do(func() {
			a.showQRDialog(qrImg, shareURL)
		})
	}()
}

func (a *App) shareToFile(s *model.Session, exercises map[string]*model.Exercise) {
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		path := writer.URI().Path()
		writer.Close()
		if !strings.HasSuffix(strings.ToLower(path), ".courtdraw") {
			path += ".courtdraw"
		}

		f, err := os.Create(path)
		if err != nil {
			a.statusBar.SetStatus(fmt.Sprintf(i18n.T("share.error"), err), 1)
			return
		}
		defer f.Close()

		if err := share.CreateBundle(f, s, exercises); err != nil {
			a.statusBar.SetStatus(fmt.Sprintf(i18n.T("share.error"), err), 1)
			return
		}
		a.statusBar.SetStatus(fmt.Sprintf(i18n.T("import.success"), s.Title), 2)
	}, a.window)

	title := s.Title
	if title == "" {
		title = "session"
	}
	d.SetFileName(title + ".courtdraw")

	dir, _ := os.UserHomeDir()
	if dir != "" {
		if listable, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
			d.SetLocation(listable)
		}
	}
	d.Show()
}

func (a *App) showQRDialog(qrImg image.Image, shareURL string) {
	img := canvas.NewImageFromImage(qrImg)
	img.SetMinSize(fyne.NewSize(256, 256))
	img.FillMode = canvas.ImageFillContain

	urlEntry := widget.NewEntry()
	urlEntry.SetText(shareURL)

	copyBtn := widget.NewButton(i18n.T("share.copy_link"), func() {
		a.window.Clipboard().SetContent(shareURL)
		a.statusBar.SetStatus(i18n.T("share.link_copied"), 2)
	})

	content := container.NewVBox(
		img,
		widget.NewLabel(i18n.T("share.qr_instructions")),
		urlEntry,
		copyBtn,
	)

	dialog.ShowCustom(i18n.T("share.qr_title"), i18n.T("dialog.cancel"), content, a.window)
}

// showImportBundleDialog checks the clipboard for a share URL. If found,
// it imports directly. Otherwise shows options (scan QR or file import).
func (a *App) showImportBundleDialog() {
	// Check clipboard first — maybe the user already scanned.
	clip := a.window.Clipboard().Content()
	if isShareURL(clip) {
		dialog.ShowConfirm(
			i18n.T("import.title"),
			i18n.T("import.found_in_clipboard"),
			func(ok bool) {
				if ok {
					a.importFromLink(clip)
				}
			},
			a.window,
		)
		return
	}

	var d dialog.Dialog

	scanBtn := widget.NewButton(i18n.T("import.scan_qr"), func() {
		d.Hide()
		a.startQRScanImport()
	})
	scanBtn.Importance = widget.HighImportance

	fileBtn := widget.NewButton(i18n.T("import.from_file"), func() {
		d.Hide()
		a.importFromFile()
	})

	content := container.NewVBox(
		scanBtn,
		widget.NewSeparator(),
		fileBtn,
	)

	d = dialog.NewCustom(i18n.T("import.title"), i18n.T("dialog.cancel"), content, a.window)
	d.Show()
}

// startQRScanImport opens the camera to take a photo of a QR code.
// When the app returns to foreground, it reads the photo and decodes the QR.
func (a *App) startQRScanImport() {
	if !openCameraForQR() {
		a.statusBar.SetStatus(i18n.T("import.camera_error"), 1)
		return
	}

	a.scanPending = true
	log.Println("QR scan: camera launched, waiting for foreground callback")
	lc := fyne.CurrentApp().Lifecycle()
	lc.SetOnEnteredForeground(func() {
		log.Println("QR scan: entered foreground, scanPending=", a.scanPending)
		if !a.scanPending {
			return
		}
		a.scanPending = false
		lc.SetOnEnteredForeground(nil)

		// Read the photo taken by the camera.
		data := readCapturedPhoto()
		log.Printf("QR scan: read photo, %d bytes", len(data))
		if data == nil || len(data) == 0 {
			cleanupPhotoURI()
			a.statusBar.SetStatus(i18n.T("import.scan_cancelled"), 1)
			return
		}

		// Decode QR from the photo.
		link, err := decodeQRFromImageBytes(data)
		if err != nil {
			log.Printf("QR scan: decode error: %v", err)
			a.statusBar.SetStatus(i18n.T("import.qr_not_found"), 1)
			return
		}
		log.Printf("QR scan: decoded link: %s", link)

		if !isShareURL(link) {
			a.statusBar.SetStatus(i18n.T("import.invalid_link"), 1)
			return
		}

		a.importFromLink(link)
	})
}

// decodeQRFromImageBytes decodes a QR code from JPEG/PNG image bytes.
// It tries multiple strategies: center-crop + various scales with bilinear interpolation.
func decodeQRFromImageBytes(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}

	// Build candidate images: full image at various sizes, then center-cropped.
	type candidate struct {
		label string
		img   image.Image
	}
	sizes := []int{600, 800, 1000, 1400}
	var candidates []candidate
	for _, sz := range sizes {
		candidates = append(candidates, candidate{fmt.Sprintf("full@%d", sz), downscaleForQR(img, sz)})
	}
	// Center crop to 60% — helps when QR is centered but photo has lots of background.
	cropped := centerCrop(img, 0.6)
	for _, sz := range sizes {
		candidates = append(candidates, candidate{fmt.Sprintf("crop60@%d", sz), downscaleForQR(cropped, sz)})
	}
	// Center crop to 40%.
	cropped40 := centerCrop(img, 0.4)
	for _, sz := range sizes {
		candidates = append(candidates, candidate{fmt.Sprintf("crop40@%d", sz), downscaleForQR(cropped40, sz)})
	}

	for _, c := range candidates {
		bmp, err := gozxing.NewBinaryBitmapFromImage(c.img)
		if err != nil {
			continue
		}
		result, err := gozxingqr.NewQRCodeReader().Decode(bmp, hints)
		if err == nil {
			log.Printf("QR decoded with %s", c.label)
			return result.GetText(), nil
		}
	}
	return "", fmt.Errorf("QR code not found")
}

// downscaleForQR resizes an image using bilinear interpolation.
func downscaleForQR(img image.Image, maxPx int) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	longest := w
	if h > longest {
		longest = h
	}
	if longest <= maxPx {
		return img
	}

	scale := float64(maxPx) / float64(longest)
	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Src, nil)
	return dst
}

// centerCrop extracts the center portion of an image (ratio 0.0–1.0).
func centerCrop(img image.Image, ratio float64) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	cropW := int(float64(w) * ratio)
	cropH := int(float64(h) * ratio)
	x0 := bounds.Min.X + (w-cropW)/2
	y0 := bounds.Min.Y + (h-cropH)/2

	dst := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	draw.Draw(dst, dst.Bounds(), img, image.Pt(x0, y0), draw.Src)
	return dst
}

// importFromFile opens a file picker for plain .courtdraw bundles.
func (a *App) importFromFile() {
	d := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		session, exercises, err := share.ExtractBundle(reader)
		if err != nil {
			a.statusBar.SetStatus(fmt.Sprintf(i18n.T("import.error"), err), 1)
			return
		}
		a.importSessionAndExercises(session, exercises)
	}, a.window)
	d.Show()
}

// isShareURL returns true if the string looks like a CourtDraw share URL.
func isShareURL(s string) bool {
	return strings.Contains(s, "#k=") && (strings.Contains(s, "tmpfiles.org") || strings.Contains(s, "file.io"))
}

// importFromLink downloads an encrypted bundle from a share URL and imports it.
func (a *App) importFromLink(link string) {
	u, err := url.Parse(link)
	if err != nil {
		a.statusBar.SetStatus(fmt.Sprintf(i18n.T("import.error"), err), 1)
		return
	}

	keyHex := u.Fragment
	if strings.HasPrefix(keyHex, "k=") {
		keyHex = keyHex[2:]
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		a.statusBar.SetStatus(i18n.T("import.invalid_link"), 1)
		return
	}

	u.Fragment = ""
	downloadURL := u.String()

	a.statusBar.SetStatus(i18n.T("import.downloading"), 0)

	go func() {
		data, err := share.Download(context.Background(), downloadURL)
		if err != nil {
			fyne.Do(func() {
				a.statusBar.SetStatus(fmt.Sprintf(i18n.T("import.error"), err), 1)
			})
			return
		}

		plaintext, err := share.Decrypt(key, data)
		if err != nil {
			fyne.Do(func() {
				a.statusBar.SetStatus(fmt.Sprintf(i18n.T("import.error"), err), 1)
			})
			return
		}

		session, exercises, err := share.ExtractBundle(bytes.NewReader(plaintext))
		if err != nil {
			fyne.Do(func() {
				a.statusBar.SetStatus(fmt.Sprintf(i18n.T("import.error"), err), 1)
			})
			return
		}

		fyne.Do(func() {
			a.importSessionAndExercises(session, exercises)
		})
	}()
}

func (a *App) importSessionAndExercises(s *model.Session, exercises map[string]*model.Exercise) {
	for _, ex := range exercises {
		if err := a.store.SaveExercise(ex); err != nil {
			log.Printf("import: save exercise %s: %v", ex.Name, err)
		}
	}

	if err := a.store.SaveSession(s); err != nil {
		a.statusBar.SetStatus(fmt.Sprintf(i18n.T("import.error"), err), 1)
		return
	}

	a.sessionTab.SetSession(s)
	a.sessionTab.SetExercises(a.buildManagedExercises())
	a.statusBar.SetStatus(fmt.Sprintf(i18n.T("import.success"), s.Title), 2)
}

func generateQR(data string) (image.Image, error) {
	qr, err := qrcode.New(data, qrcode.High)
	if err != nil {
		return nil, err
	}
	return qr.Image(512), nil
}
