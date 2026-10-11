package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const prefix = "github.com/foods-platform/foods-backend/internal/"

// Enforce dependencies across every module, shared primitive and HTTP router.
func TestModuleBoundaries(t *testing.T) {
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(path)
		if !strings.Contains(rel, "/modules/") && !strings.Contains(rel, "/platform/") && !strings.Contains(rel, "/transport/") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		contracts := strings.Contains(rel, "/contracts/")
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
			// Published contracts may reference their own domain but never a
			// concrete adapter, HTTP transport, use-case implementation or SQL driver.
			if contracts && (strings.Contains(target, "/application") || strings.Contains(target, "/infrastructure") ||
				strings.Contains(target, "/transport") || target == "net/http" || target == "database/sql" ||
				strings.HasPrefix(target, "github.com/jackc/") || strings.HasPrefix(target, prefix+"platform/database")) {
				violation = true
			}
			if application && (strings.Contains(target, "/infrastructure/") || strings.Contains(target, "/transport/") || target == "net/http" || target == "database/sql" || strings.HasPrefix(target, "github.com/jackc/") || strings.HasSuffix(target, "/platform/httpapi")) {
				violation = true
			}
			if transport && (strings.Contains(target, "/infrastructure/") || strings.HasPrefix(target, "github.com/jackc/") || target == "database/sql") {
				violation = true
			}
			// Application and HTTP layers must reach persistence only through their ports.
			if (application || transport || strings.Contains(rel, "/platform/httpapi/")) && strings.HasPrefix(target, prefix+"platform/database") {
				violation = true
			}
			if infrastructure && (strings.Contains(target, "/transport/") || target == "net/http" || strings.HasSuffix(target, "/platform/httpapi")) {
				violation = true
			}
			if violation {
				t.Errorf("%s imports forbidden dependency %s", rel, target)
			}
		}
		if domain || contracts || application || transport || strings.Contains(rel, "/platform/httpapi/") {
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

func TestLegacyPackageRetired(t *testing.T) {
	if _, err := os.Stat("../legacy"); !os.IsNotExist(err) {
		t.Fatalf("legacy must not be reintroduced: %v", err)
	}
}
