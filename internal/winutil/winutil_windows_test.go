//go:build windows

package winutil

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGetFileDescriptionAndExtractIconOnSystemBinaries(t *testing.T) {
	winDir := os.Getenv("SystemRoot")
	if winDir == "" {
		winDir = `C:\Windows`
	}

	candidates := []string{
		filepath.Join(winDir, "System32", "notepad.exe"),
		filepath.Join(winDir, "System32", "cmd.exe"),
		filepath.Join(winDir, "regedit.exe"),
		filepath.Join(winDir, "explorer.exe"),
	}

	checked := 0
	for _, exe := range candidates {
		if _, err := os.Stat(exe); err != nil {
			continue
		}
		checked++

		desc := GetFileDescription(exe)
		if desc == "" {
			t.Errorf("expected non-empty FileDescription for %s", exe)
		}

		pngBytes, err := ExtractIconPNG(exe, 64)
		if err != nil {
			t.Fatalf("ExtractIconPNG(%s, 64) failed: %v", exe, err)
		}
		img, err := png.Decode(bytes.NewReader(pngBytes))
		if err != nil {
			t.Fatalf("decoded icon for %s is not valid PNG: %v", exe, err)
		}
		b := img.Bounds()
		if b.Dx() == 0 || b.Dy() == 0 {
			t.Fatalf("invalid icon dimensions for %s: %v", exe, b)
		}

		// 确认图标中存在不透明可见像素，避免掩码/Alpha 处理异常导致图标全透明。
		visiblePixels := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a > 0 {
					visiblePixels++
				}
			}
		}
		if visiblePixels == 0 {
			t.Fatalf("icon for %s has 0 visible pixels (fully transparent)", exe)
		}
		t.Logf("%s -> desc=%q icon=%dx%d visible=%d/%d (%d bytes)",
			filepath.Base(exe), desc, b.Dx(), b.Dy(), visiblePixels, b.Dx()*b.Dy(), len(pngBytes))
	}
	if checked == 0 {
		t.Skip("no Windows system binaries found")
	}
}
