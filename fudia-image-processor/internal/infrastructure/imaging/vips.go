package imaging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/foods-platform/fudia-image-processor/internal/application"
	"github.com/foods-platform/fudia-image-processor/internal/domain"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type Vips struct{ Diagnostics func(string) }

func Validate(content []byte, record domain.ImageRecord) error {
	if int64(len(content)) > domain.MaxImageBytes || int64(len(content)) != record.SizeBytes {
		return application.ErrInvalidImage
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(content))
	expected := map[string]string{"image/jpeg": "jpeg", "image/png": "png", "image/webp": "webp"}[record.ContentType]
	dimensionsMatch := config.Width == record.Width && config.Height == record.Height
	if format == "jpeg" && config.Width == record.Height && config.Height == record.Width {
		dimensionsMatch = true
	}
	if err != nil || format != expected || !dimensionsMatch || config.Width < 1 || config.Height < 1 || config.Width > 4000 || config.Height > 4000 {
		return application.ErrInvalidImage
	}
	// Decode the whole first frame too: a valid header alone does not prove a valid image.
	if _, _, err = image.Decode(bytes.NewReader(content)); err != nil {
		return application.ErrInvalidImage
	}
	return nil
}
func (v Vips) Transform(ctx context.Context, content []byte, record domain.ImageRecord) (map[string][]byte, error) {
	if err := Validate(content, record); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "fudia-image-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	input := filepath.Join(directory, "source")
	if err = os.WriteFile(input, content, 0600); err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, variant := range []struct {
		name string
		edge int
	}{{"thumbnail", 200}, {"medium", 600}} {
		output := filepath.Join(directory, variant.name+".webp")
		commandCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		// No shell, no user-supplied paths/options. Auto-orient; sRGB; strip metadata; never upscale.
		cmd := exec.CommandContext(commandCtx, "vipsthumbnail", input, "--size", strconv.Itoa(variant.edge)+">", "--export-profile", "srgb", "-o", output+"[Q=82,strip]")
		cmd.Env = append(os.Environ(), "VIPS_CONCURRENCY=1")
		var diagnostics bytes.Buffer
		if v.Diagnostics != nil {
			cmd.Stderr = &diagnostics
		}
		err = cmd.Run()
		contextErr := commandCtx.Err()
		cancel()
		if err != nil {
			if v.Diagnostics != nil {
				v.Diagnostics(diagnostics.String())
			}
			if errors.Is(err, exec.ErrNotFound) || contextErr != nil {
				return nil, fmt.Errorf("image converter unavailable: %w", err)
			}
			return nil, fmt.Errorf("image conversion failed: %w", err)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			return nil, err
		}
		config, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || format != "webp" || config.Width > variant.edge || config.Height > variant.edge {
			return nil, errors.New("invalid optimized output")
		}
		out[variant.name] = data
	}
	return out, nil
}
