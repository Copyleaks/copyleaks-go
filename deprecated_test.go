package copyleaks

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestDeprecatedMarkers checks that the fields that the API reference deprecates carry
// a "Deprecated:" paragraph, which go vet, gopls and linters report on use.
func TestDeprecatedMarkers(t *testing.T) {
	want := map[string]bool{"PDF.Version": false, "PDFVersion": false, "AITextDetectionResult.Probability": false}
	fset := token.NewFileSet()
	for _, file := range []string{"scan_properties.go", "ai_text.go"} {
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			spec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			check := func(name string, doc *ast.CommentGroup) {
				if _, tracked := want[name]; tracked && doc != nil && strings.Contains(doc.Text(), "\nDeprecated: ") {
					want[name] = true
				}
			}
			if gen := typeDoc(f, spec); gen != nil {
				check(spec.Name.Name, gen)
			}
			if st, ok := spec.Type.(*ast.StructType); ok {
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						check(spec.Name.Name+"."+name.Name, field.Doc)
					}
				}
			}
			return true
		})
	}
	for name, ok := range want {
		if !ok {
			t.Errorf("%s has no Deprecated: paragraph", name)
		}
	}
}

// typeDoc returns the doc comment of a type, which go/parser attaches to the type spec
// or, for a lone type declaration, to the declaration.
func typeDoc(f *ast.File, spec *ast.TypeSpec) *ast.CommentGroup {
	if spec.Doc != nil {
		return spec.Doc
	}
	for _, decl := range f.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && len(gen.Specs) == 1 && gen.Specs[0] == spec {
			return gen.Doc
		}
	}
	return nil
}
