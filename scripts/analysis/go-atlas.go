// Standalone standard-library parser used by code-atlas.mjs. Run from backend/.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
)

type function struct {
	Name     string `json:"name"`
	Line     int    `json:"line"`
	End      int    `json:"end"`
	Branches int    `json:"branches"`
}

type result struct {
	Path      string     `json:"path"`
	Functions []function `json:"functions"`
	Imports   []string   `json:"imports"`
	Error     string     `json:"error,omitempty"`
}

func main() {
	var paths []string
	if err := json.NewDecoder(os.Stdin).Decode(&paths); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	results := make([]result, 0, len(paths))
	for _, path := range paths {
		r := result{Path: path, Functions: []function{}, Imports: []string{}}
		fs := token.NewFileSet()
		file, err := parser.ParseFile(fs, path, nil, 0)
		if err != nil {
			r.Error = err.Error()
			results = append(results, r)
			continue
		}
		for _, imp := range file.Imports {
			value, _ := strconv.Unquote(imp.Path.Value)
			r.Imports = append(r.Imports, value)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			var body *ast.BlockStmt
			name := ""
			switch n := node.(type) {
			case *ast.FuncDecl:
				name, body = n.Name.Name, n.Body
				if n.Recv != nil && len(n.Recv.List) > 0 {
					recv := n.Recv.List[0].Type
					if star, ok := recv.(*ast.StarExpr); ok {
						recv = star.X
					}
					if id, ok := recv.(*ast.Ident); ok {
						name = id.Name + "." + name
					}
				}
			case *ast.FuncLit:
				name, body = "匿名函数", n.Body
			}
			if body != nil {
				branches := 1
				ast.Inspect(body, func(child ast.Node) bool {
					switch n := child.(type) {
					case *ast.FuncLit:
						return false
					case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
						branches++
					case *ast.CaseClause:
						if len(n.List) > 0 {
							branches++
						}
					case *ast.CommClause:
						if n.Comm != nil {
							branches++
						}
					case *ast.BinaryExpr:
						if n.Op == token.LAND || n.Op == token.LOR {
							branches++
						}
					}
					return true
				})
				r.Functions = append(r.Functions, function{name, fs.Position(node.Pos()).Line, fs.Position(node.End() - 1).Line, branches})
			}
			return true
		})
		results = append(results, r)
	}
	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
