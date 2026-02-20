package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

// ParsedFile holds declarations extracted from a source file.
type ParsedFile struct {
	NeedsTime bool   // Whether "time" import is required
	Decls     []Decl // Ordered list of declarations
}

// Decl is a tagged union for emittable declarations.
type Decl interface {
	declMarker()
}

// TypeAliasDecl represents `type WorkflowStatus string`.
type TypeAliasDecl struct {
	Doc        string // Doc comment text (with // prefix and newlines)
	Name       string
	Underlying string // e.g., "string"
}

func (*TypeAliasDecl) declMarker() {}

// ConstBlockDecl represents a `const (...)` group.
type ConstBlockDecl struct {
	Doc    string // Doc comment above the block
	Consts []ConstDecl
}

func (*ConstBlockDecl) declMarker() {}

// ConstDecl represents a single constant within a block.
type ConstDecl struct {
	Doc     string // Doc/section comment preceding this const
	Name    string
	Type    string // May be empty for untyped consts
	Value   string // e.g., `"draft"`
	Comment string // Trailing inline comment
}

// StructDecl represents `type Foo struct { ... }`.
type StructDecl struct {
	Doc    string
	Name   string
	Fields []FieldDecl
}

func (*StructDecl) declMarker() {}

// FieldDecl represents a single struct field.
type FieldDecl struct {
	Name    string
	Type    string // Rendered type expression
	Tag     string // Full struct tag including backticks
	Comment string // Trailing inline comment
}

// ParseSourceFile parses a Go source file and extracts declarations per the given rule.
func ParseSourceFile(path string, rule FileRule) (*ParsedFile, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	pf := &ParsedFile{}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				pf.extractTypes(fset, d, rule)
			case token.CONST:
				pf.extractConsts(fset, d, file, rule)
			}
		// *ast.FuncDecl — skip all functions/methods
		}
	}

	return pf, nil
}

func (pf *ParsedFile) extractTypes(fset *token.FileSet, gd *ast.GenDecl, rule FileRule) {
	for _, spec := range gd.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		name := ts.Name.Name
		if rule.ExcludeTypes[name] {
			continue
		}

		doc := docText(gd, ts)

		switch t := ts.Type.(type) {
		case *ast.StructType:
			sd := &StructDecl{Doc: doc, Name: name}
			excludeFields := toSet(rule.ExcludeFields[name])

			for _, field := range t.Fields.List {
				if len(field.Names) == 0 {
					// Embedded field — skip (we don't carry BaseResource embedding)
					continue
				}
				fname := field.Names[0].Name
				if excludeFields[fname] {
					continue
				}

				typeStr := renderExpr(fset, field.Type)
				if strings.Contains(typeStr, "time.") {
					pf.NeedsTime = true
				}

				fd := FieldDecl{
					Name: fname,
					Type: typeStr,
				}
				if field.Tag != nil {
					fd.Tag = field.Tag.Value
				}
				if field.Comment != nil {
					fd.Comment = strings.TrimSpace(field.Comment.Text())
				}

				sd.Fields = append(sd.Fields, fd)
			}
			pf.Decls = append(pf.Decls, sd)

		case *ast.Ident:
			// type alias like `type WorkflowStatus string`
			pf.Decls = append(pf.Decls, &TypeAliasDecl{
				Doc:        doc,
				Name:       name,
				Underlying: t.Name,
			})

		default:
			// Other type expressions — render as-is
			pf.Decls = append(pf.Decls, &TypeAliasDecl{
				Doc:        doc,
				Name:       name,
				Underlying: renderExpr(fset, ts.Type),
			})
		}
	}
}

func (pf *ParsedFile) extractConsts(fset *token.FileSet, gd *ast.GenDecl, file *ast.File, rule FileRule) {
	cb := &ConstBlockDecl{
		Doc: genDeclDoc(gd),
	}

	// Track current iota type to detect and skip excluded-type const blocks.
	currentType := ""

	for _, spec := range gd.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// Update current type if spec has explicit type
		if vs.Type != nil {
			currentType = renderExpr(fset, vs.Type)
		}

		// Skip consts whose type is in ExcludeTypes
		if rule.ExcludeTypes[currentType] {
			continue
		}

		for i, name := range vs.Names {
			if !name.IsExported() {
				continue
			}

			cd := ConstDecl{
				Name: name.Name,
			}

			// Collect doc/section comments preceding this const
			cd.Doc = constDoc(fset, file, vs, spec)

			if vs.Type != nil {
				cd.Type = renderExpr(fset, vs.Type)
			}

			if i < len(vs.Values) {
				cd.Value = renderExpr(fset, vs.Values[i])
			}

			if vs.Comment != nil {
				cd.Comment = strings.TrimSpace(vs.Comment.Text())
			}

			cb.Consts = append(cb.Consts, cd)
		}
	}

	if len(cb.Consts) > 0 {
		pf.Decls = append(pf.Decls, cb)
	}
}

// docText extracts the doc comment for a TypeSpec, preferring the spec's own Doc
// and falling back to the parent GenDecl's Doc (for single-spec type declarations).
func docText(gd *ast.GenDecl, ts *ast.TypeSpec) string {
	if ts.Doc != nil {
		return formatCommentGroup(ts.Doc)
	}
	if gd.Doc != nil && len(gd.Specs) == 1 {
		return formatCommentGroup(gd.Doc)
	}
	return ""
}

func genDeclDoc(gd *ast.GenDecl) string {
	if gd.Doc != nil {
		return formatCommentGroup(gd.Doc)
	}
	return ""
}

// constDoc collects section/doc comments that appear before a const spec.
// These are free-floating comments in the const block (like "// Execution-level events").
func constDoc(_ *token.FileSet, _ *ast.File, vs *ast.ValueSpec, _ ast.Spec) string {
	if vs.Doc != nil {
		return formatCommentGroup(vs.Doc)
	}
	return ""
}

func formatCommentGroup(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	var lines []string
	for _, c := range cg.List {
		lines = append(lines, c.Text)
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderExpr(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, expr)
	return buf.String()
}

func toSet(ss []string) map[string]bool {
	if len(ss) == 0 {
		return nil
	}
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
