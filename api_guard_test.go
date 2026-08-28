package ariadne_test

// The public API is the whole product to anyone importing this module, so it is
// guarded like one: this test lists every exported name of package ariadne and
// compares it against testdata/public_api.txt. Adding, renaming or removing a
// name fails until the list is updated on purpose, which is what stops the
// surface from growing by accident.
//
// Review the diff, then re-baseline deliberately with:
//
//	go test -run TestPublicAPISurfaceIsAnIntent -update

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const publicAPILayoutPath = "testdata/public_api.txt"

var updatePublicAPILayout = flag.Bool("update", false, "rewrite the golden public API list")

func TestPublicAPISurfaceIsAnIntent(t *testing.T) {
	got := publicAPISurface(t)

	if *updatePublicAPILayout {
		require.NoError(t, os.WriteFile(publicAPILayoutPath, []byte(got+"\n"), 0o644))
		t.Logf("rewrote %s with %d names", publicAPILayoutPath, len(strings.Split(got, "\n")))
		return
	}

	wantBytes, err := os.ReadFile(publicAPILayoutPath)
	require.NoError(t, err, "run the test with -update to create it")
	require.Equal(t, strings.TrimSpace(string(wantBytes)), got,
		"the public API changed; review the diff and re-run with -update if it is intended")
}

// publicAPISurface renders each exported declaration as one sorted line, so a
// diff shows exactly which name appeared or disappeared.
func publicAPISurface(t *testing.T) string {
	t.Helper()

	fileSet := token.NewFileSet()
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fileSet, entry.Name(), nil, parser.SkipObjectResolution)
		require.NoError(t, err)
		if file.Name.Name != "ariadne" {
			continue
		}
		for _, decl := range file.Decls {
			names = append(names, exportedNames(decl)...)
		}
	}
	sort.Strings(names)
	return strings.Join(dedupe(names), "\n")
}

func exportedNames(decl ast.Decl) []string {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		if !decl.Name.IsExported() {
			return nil
		}
		if recv := receiverName(decl); recv != "" {
			return []string{"func " + recv + "." + decl.Name.Name}
		}
		return []string{"func " + decl.Name.Name}
	case *ast.GenDecl:
		keyword := ""
		switch decl.Tok {
		case token.CONST:
			keyword = "const"
		case token.TYPE:
			keyword = "type"
		case token.VAR:
			keyword = "var"
		default:
			return nil
		}
		var names []string
		for _, spec := range decl.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				if spec.Name.IsExported() {
					names = append(names, keyword+" "+spec.Name.Name)
				}
			case *ast.ValueSpec:
				for _, ident := range spec.Names {
					if ident.IsExported() {
						names = append(names, keyword+" "+ident.Name)
					}
				}
			}
		}
		return names
	default:
		return nil
	}
}

func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	switch recv := fn.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		return typeName(recv.X)
	case *ast.Ident:
		return recv.Name
	default:
		return ""
	}
}

func typeName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.IndexExpr:
		if ident, ok := expr.X.(*ast.Ident); ok {
			return ident.Name
		}
		return ""
	case *ast.IndexListExpr:
		if ident, ok := expr.X.(*ast.Ident); ok {
			return ident.Name
		}
		return ""
	default:
		return ""
	}
}

func dedupe(sorted []string) []string {
	var out []string
	for i, name := range sorted {
		if i > 0 && name == sorted[i-1] {
			continue
		}
		out = append(out, name)
	}
	return out
}
