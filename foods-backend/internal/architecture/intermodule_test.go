package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Module-to-module communication may use only a published contracts package.
// Audit is a shared interface-only capability, not a business module.
// Application, domain, transport and persistence internals are private to their owner.
func TestOnlyPublishedIntermoduleContracts(t *testing.T) {
	err := filepath.WalkDir("../modules", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(path)
		parts := strings.SplitN(rel, "/modules/", 2)
		if len(parts) != 2 {
			return nil
		}
		owner := strings.SplitN(parts[1], "/", 2)[0]
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			target, _ := strconv.Unquote(imp.Path.Value)
			const modules = prefix + "modules/"
			if !strings.HasPrefix(target, modules) {
				continue
			}
			imported := strings.Split(strings.TrimPrefix(target, modules), "/")
			if len(imported) < 2 || imported[0] == owner {
				continue
			}
			if imported[0] == "audit" && len(imported) == 2 && imported[1] == "application" {
				continue
			}
			if imported[1] != "contracts" {
				t.Errorf("%s directly imports another module's internals: %s; use a published contract", rel, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
