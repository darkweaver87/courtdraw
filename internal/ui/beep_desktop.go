//go:build !android && !ios

package ui

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

var (
	otoCtx     *oto.Context
	otoOnce    sync.Once
	errOtoInit error
)

func initAudio() {
	otoOnce.Do(func() {
		op := &oto.NewContextOptions{
			SampleRate:   44100,
			ChannelCount: 1,
			Format:       oto.FormatSignedInt16LE,
		}
		var ready chan struct{}
		otoCtx, ready, errOtoInit = oto.NewContext(op)
		if errOtoInit == nil {
			<-ready
		}
	})
}

// systemBeep plays a short 200ms sine tone at 880Hz.
func systemBeep() {
	initAudio()
	if errOtoInit != nil || otoCtx == nil {
		return
	}

	const (
		sampleRate = 44100
		freq       = 880.0 // Hz (A5)
		durationMs = 200
	)
	numSamples := sampleRate * durationMs / 1000
	buf := make([]byte, numSamples*2)
	fadeLen := sampleRate * 10 / 1000 // 10ms fade

	for i := range numSamples {
		t := float64(i) / float64(sampleRate)
		env := 1.0
		if i < fadeLen {
			env = float64(i) / float64(fadeLen)
		} else if numSamples-i < fadeLen {
			env = float64(numSamples-i) / float64(fadeLen)
		}
		sample := int16(env * 0.4 * math.Sin(2*math.Pi*freq*t) * 32767)
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(sample)) //nolint:gosec // intentional int16→uint16 for PCM
	}

	player := otoCtx.NewPlayer(bytes.NewReader(buf))
	go func() {
		player.Play()
		time.Sleep(time.Duration(durationMs+50) * time.Millisecond)
		_ = player.Close()
	}()
}
