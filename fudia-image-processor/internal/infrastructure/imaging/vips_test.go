package imaging

import (
	"bytes"
	"context"
	"errors"
	"github.com/foods-platform/fudia-image-processor/internal/application"
	"github.com/foods-platform/fudia-image-processor/internal/domain"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os/exec"
	"testing"
)

func testPNG(width, height int) ([]byte, domain.ImageRecord) {
	var buffer bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, width, height))
	picture.Set(0, 0, color.RGBA{R: 250, A: 255})
	_ = png.Encode(&buffer, picture)
	data := buffer.Bytes()
	return data, domain.ImageRecord{ContentType: "image/png", SizeBytes: int64(len(data)), Width: width, Height: height}
}
func TestValidateActualImageAndDimensions(t *testing.T) {
	data, record := testPNG(800, 400)
	if err := Validate(data, record); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"format", "dimensions", "size", "truncated", "fake"} {
		t.Run(kind, func(t *testing.T) {
			content := data
			metadata := record
			switch kind {
			case "format":
				metadata.ContentType = "image/jpeg"
			case "dimensions":
				metadata.Width = 999
			case "size":
				metadata.SizeBytes++
			case "truncated":
				content = data[:len(data)/2]
				metadata.SizeBytes = int64(len(content))
			case "fake":
				content = []byte("<svg/>")
				metadata.SizeBytes = int64(len(content))
			}
			if !errors.Is(Validate(content, metadata), application.ErrInvalidImage) {
				t.Fatal("invalid image accepted")
			}
		})
	}
	huge, metadata := testPNG(4001, 1)
	if err := Validate(huge, metadata); !errors.Is(err, application.ErrInvalidImage) {
		t.Fatal("oversized resolution accepted")
	}
	var jpegData bytes.Buffer
	_ = jpeg.Encode(&jpegData, image.NewRGBA(image.Rect(0, 0, 100, 50)), nil)
	if err := Validate(jpegData.Bytes(), domain.ImageRecord{ContentType: "image/jpeg", SizeBytes: int64(jpegData.Len()), Width: 50, Height: 100}); err != nil {
		t.Fatal("browser oriented JPEG rejected")
	}
}
func TestVipsWebPVariantsNoUpscaling(t *testing.T) {
	if _, err := exec.LookPath("vipsthumbnail"); err != nil {
		t.Skip("real converter runs in the image-processor Docker verification")
	}
	for _, width := range []int{800, 80} {
		data, record := testPNG(width, width/2)
		variants, err := (Vips{Diagnostics: func(message string) { t.Log(message) }}).Transform(context.Background(), data, record)
		if err != nil {
			t.Fatal(err)
		}
		if len(variants) != 2 {
			t.Fatal("unexpected variants")
		}
		for name, output := range variants {
			config, format, err := image.DecodeConfig(bytes.NewReader(output))
			if err != nil || format != "webp" {
				t.Fatal("not WebP")
			}
			edge := 200
			if name == "medium" {
				edge = 600
			}
			if width < edge {
				edge = width
			}
			if config.Width != edge || config.Height != edge/2 {
				t.Fatalf("%s: %dx%d", name, config.Width, config.Height)
			}
		}
	}
}
