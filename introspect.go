package introspect

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/sridharv/fail"
)

func NewPackage(path string) (*PackageBuilder, error) {
	pkg, err := build.Import(path, ".", build.AllowBinary)
	if err != nil {
		return nil, err
	}
	return &PackageBuilder{pkg}, nil
}

type PackageBuilder struct {
	pkg *build.Package
}

func (p *PackageBuilder) Build() (*Package, error) {
	files := make([]File, 0)
	for _, name := range p.pkg.GoFiles {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		fileName := filepath.Join(p.pkg.Dir, name)
		builder, err := NewFileBuilder(fileName)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", fileName, err)
		}

		file, err := builder.Build()
		if err != nil {
			return nil, fmt.Errorf("%s: %v", fileName, err)
		}
		files = append(files, *file)
	}
	return &Package{Files: files}, nil
}

type Package struct {
	Files []File
}

type File struct {
	Types
	Name string
}

type FileBuilder struct {
	File
	parsed *ast.File
}

func NewFileBuilder(fileName string) (*FileBuilder, error) {
	parsedFile, err := parser.ParseFile(token.NewFileSet(), fileName, nil, 0)
	if err != nil {
		return nil, err
	}
	return &FileBuilder{
		parsed: parsedFile,
		File: File{
			Types: Types{
				Interfaces: []Interface{},
				Structs:    []Struct{},
			},
			Name: fileName,
		},
	}, nil
}

func (f *FileBuilder) Build() (*File, error) {
	var err error = nil
	func() {
		defer fail.Using(func(args ...interface{}) { err = errors.New(fmt.Sprint(args...)) })
		ast.Inspect(f.parsed, f.inspect)
	}()
	return &f.File, err
}

func (f *FileBuilder) inspect(node ast.Node) bool {
	switch typed := node.(type) {
	case *ast.TypeSpec:
		name := typed.Name.Name
		switch subType := typed.Type.(type) {
		case *ast.StructType:
			f.Struct(name).addAll(subType.Fields)
		case *ast.InterfaceType:
			f.Interface(name).addAll(subType.Methods)
		}
	}
	return true
}

func fieldName(ident ...*ast.Ident) string {
	if len(ident) == 0 {
		return ""
	}
	fail.If(len(ident) != 1, "more than one identifier", ident)
	return ident[0].Name
}

func newField(name string, field ast.Expr) Field {
	switch typed := field.(type) {
	case *ast.FuncType:
		fn := &Func{name, make([]Field, 0), make([]Field, 0)}
		fn.Params.addAll(typed.Params)
		fn.Results.addAll(typed.Results)
		return fn
	case *ast.ChanType:
		return Chan{Dir: typed.Dir, Type: newField("", typed.Value)}
	case *ast.Ident:
		return TypedField{name, typed.Name}
	default:
		fail.If(false, fmt.Sprintf("unsupported type:%T\n", typed))
		return nil
	}
}

type Field interface {
}

type Func struct {
	Name    string
	Params  FieldList
	Results FieldList
}

type Chan struct {
	Dir  ast.ChanDir
	Type Field
}

type TypedField struct {
	Name string
	Type string
}

type FieldList []Field

func NewFieldList() *FieldList { return &FieldList{} }

func (f *FieldList) addAll(list *ast.FieldList) {
	if f == nil || list == nil {
		return
	}
	for _, field := range list.List {
		parsed := newField(fieldName(field.Names...), field.Type)
		if parsed == nil {
			continue
		}
		*f = append(*f, parsed)
	}
}


type Interface struct {
	*FieldList
	Name string
}

type Struct struct {
	*FieldList
	Name string
}

type Types struct {
	Interfaces []Interface
	Structs    []Struct
}

func (w *Types) Interface(name string) *FieldList {
	f := NewFieldList()
	w.Interfaces = append(w.Interfaces, Interface{f, name})
	return f
}

func (w *Types) Struct(name string) *FieldList {
	f := NewFieldList()
	w.Structs = append(w.Structs, Struct{f, name})
	return f
}
