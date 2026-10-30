package imaging

import (
	"bytes"
	"context"
	"image"
	"os/exec"
	"testing"
)

func TestBrandingProfilesPreserveAlphaAndReadableMenu(t *testing.T) {
	if _, err := exec.LookPath("vipsthumbnail"); err != nil {
		t.Skip("libvips verification runs in Docker")
	}
	for _, purpose := range []string{"restaurant_logo", "virtual_menu"} {
		data, record := testPNG(3000, 1500)
		record.Purpose = purpose
		variants, err := (Vips{Diagnostics: func(s string) { t.Log(s) }}).Transform(context.Background(), data, record)
		if err != nil {
			t.Fatal(err)
		}
		expected := 600
		name, format := "logo", "png"
		if purpose == "virtual_menu" {
			expected = 2400
			name, format = "delivery", "jpeg"
		}
		picture, actual, err := image.Decode(bytes.NewReader(variants[name]))
		if err != nil || actual != format || picture.Bounds().Dx() != expected {
			t.Fatalf("profile %s %s %v", purpose, actual, err)
		}
		if name == "logo" {
			_, _, _, alpha := picture.At(expected-1, 0).RGBA()
			if alpha != 0 {
				t.Fatal("logo lost transparency")
			}
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(variants["medium"]))
		if err != nil || config.Width != expected {
			t.Fatal("incorrect UI size", err)
		}
	}
}
