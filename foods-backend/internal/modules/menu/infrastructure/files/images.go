package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Images struct{ Directory string }

func (s Images) Save(_ context.Context, id, extension string, content io.Reader) (string, error) {
	if id == "" || filepath.Base(id) != id || strings.ContainsAny(id, "/\\") {
		return "", fmt.Errorf("invalid product id")
	}
	if err := os.MkdirAll(s.Directory, 0755); err != nil {
		return "", err
	}
	name := id + extension
	dst, err := os.Create(filepath.Join(s.Directory, name))
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(dst, content)
	closeErr := dst.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return "/uploads/products/" + name, nil
}
