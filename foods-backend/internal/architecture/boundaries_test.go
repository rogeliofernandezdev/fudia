package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const prefix = "github.com/foods-platform/foods-backend/internal/"

// Enforce real dependencies, not directory naming conventions. Legacy is isolated
// explicitly and cannot leak into the extracted modules or platform primitives.
func TestModuleBoundaries(t *testing.T) {
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(path)
		if !strings.Contains(rel, "/modules/") && !strings.Contains(rel, "/platform/") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		domain := strings.Contains(rel, "/domain/")
		application := strings.Contains(rel, "/application/")
		transport := strings.Contains(rel, "/transport/")
		infrastructure := strings.Contains(rel, "/infrastructure/")
		for _, imp := range f.Imports {
			target, _ := strconv.Unquote(imp.Path.Value)
			violation := strings.HasPrefix(target, prefix+"legacy/")
			if strings.Contains(rel, "/platform/") && strings.HasPrefix(target, prefix+"modules/") {
				violation = true
			}
			if domain && (strings.Contains(target, ".") || target == "net/http" || target == "database/sql" || strings.HasPrefix(target, prefix)) {
				violation = true
			}
			if application && (strings.Contains(target, "/infrastructure/") || strings.Contains(target, "/transport/") || target == "net/http" || target == "database/sql" || strings.HasPrefix(target, "github.com/jackc/") || strings.HasSuffix(target, "/platform/httpapi")) {
				violation = true
			}
			if transport && (strings.Contains(target, "/infrastructure/") || strings.HasPrefix(target, "github.com/jackc/") || target == "database/sql") {
				violation = true
			}
			if infrastructure && (strings.Contains(target, "/transport/") || target == "net/http" || strings.HasSuffix(target, "/platform/httpapi")) {
				violation = true
			}
			if violation {
				t.Errorf("%s imports forbidden dependency %s", rel, target)
			}
		}
		if domain || application || transport {
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				v, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				v = strings.ToUpper(strings.TrimSpace(v))
				for _, sql := range []string{"SELECT ", "INSERT INTO ", "UPDATE ", "DELETE FROM "} {
					if strings.HasPrefix(v, sql) {
						t.Errorf("SQL outside infrastructure: %s", rel)
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
