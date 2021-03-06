package internal

import (
	"fmt"
	"go/ast"
	"go/token"
)

// ArgsToFunctionArgs return the list of statements
func ArgsToFunctionArgs(argType string, name string, fields []*FieldDef) []ast.Stmt {
	stmts := make([]ast.Stmt, 0)

	for index, f := range fields {
		resultName := fmt.Sprintf("%s%v", name, index)

		stmts = append(stmts, &ast.AssignStmt{
			Lhs: []ast.Expr{
				NewIdentObj(resultName),
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   NewIdentObjVar(name),
						Sel: NewIdent("Get"),
					},
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: fmt.Sprintf(`"%s"`, f.Name),
						},
					},
				},
			},
		})
		stmts = append(stmts, argumentToVariable(resultName, fmt.Sprintf("%s%v", argType, index), f))
	}

	return stmts
}

func ifArgumentValueIsNotNil(variable string, stmt ast.Stmt) ast.Stmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   NewIdentObjVar(variable),
					Sel: NewIdent("Value"),
				},
			},
			Op: token.NEQ,
			Y:  NewIdent("nil"),
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{stmt}},
	}
}

//nolint:funlen
func appendResultsStatements(results []*FieldDef) []ast.Stmt {
	stmts := make([]ast.Stmt, 0)
	stmts = append(stmts, &ast.AssignStmt{
		Lhs: []ast.Expr{NewIdentObjVar("beyondResults")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   NewIdentObjVar(varBeyondContext),
					Sel: NewIdent("Results"),
				},
			},
		},
	})
	returnExpr := make([]ast.Expr, len(results))

	for i := range results {
		result := results[i]

		stmts = append(stmts, &ast.AssignStmt{
			Lhs: []ast.Expr{NewIdentObjVar(fmt.Sprintf("beyondResult%v", i))},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   NewIdentObjVar("beyondResults"),
						Sel: NewIdent("At"),
					},
					Args: []ast.Expr{
						NewIdent(fmt.Sprintf("%v", i)),
					},
				},
			},
		})

		stmts = append(stmts, &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Specs: []ast.Spec{
					&ast.ValueSpec{

						Names: []*ast.Ident{
							NewIdentObj(fmt.Sprintf("result%v", i)),
						},
						Type: result.Kind,
					},
				},
				Tok: token.VAR,
			},
		})

		stmts = append(stmts, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   NewIdentObjVar(fmt.Sprintf("beyondResult%v", i)),
						Sel: NewIdent("Value"),
					},
				},
				Op: token.NEQ,
				Y:  NewIdentObj("nil"),
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{
							NewIdentObj(fmt.Sprintf("result%v", i)),
						},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{
							&ast.TypeAssertExpr{
								X: &ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X:   NewIdentObjVar(fmt.Sprintf("beyondResult%v", i)),
										Sel: NewIdent("Value"),
									},
								},
								Type: result.Kind,
							},
						},
					},
				},
			},
		})

		returnExpr[i] = NewIdent(fmt.Sprintf("result%v", i))
	}

	return append(stmts, &ast.ReturnStmt{
		Results: returnExpr,
	})
}

/**
var result1 error
                if beyondResult1.Value()!=nil{
                        result1=beyondResult0.Value().(error)
                }
*/
// IfAdviceIsCompleted add statements if advice is completed
func IfAdviceIsCompleted(results []*FieldDef) ast.Stmt {
	stmts := make([]ast.Stmt, 0)

	if len(results) == 0 {
		stmts = append(stmts, &ast.ReturnStmt{})
	} else {
		stmts = append(stmts, appendResultsStatements(results)...)
	}

	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   NewIdentObjVar(varBeyondContext),
					Sel: NewIdent("IsCompleted"),
				},
			},
			Op: token.EQL,
			Y:  NewIdent("true"),
		},
		Body: &ast.BlockStmt{List: stmts},
	}
}

func argumentToVariable(variable string, fieldName string, field *FieldDef) ast.Stmt {
	kind := field.Kind
	if ell, ok := kind.(*ast.Ellipsis); ok {
		kind = &ast.ArrayType{
			Elt: ell.Elt,
		}
	}

	assigmentStmt := &ast.AssignStmt{
		Tok: token.ASSIGN,
		Lhs: []ast.Expr{
			NewIdentObjVar(fieldName),
		},
		Rhs: []ast.Expr{
			&ast.TypeAssertExpr{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   NewIdentObjVar(variable),
						Sel: NewIdent("Value"),
					},
				},
				Type: kind,
			},
		},
	}

	return ifArgumentValueIsNotNil(variable, assigmentStmt)
}

// ReturnValuesStmt return the list of statements
func ReturnValuesStmt(fields []*FieldDef) ast.Stmt {
	results := make([]ast.Expr, len(fields))

	for index, field := range fields {
		results[index] = NewIdentObjVar(field.Name)
	}

	return &ast.ReturnStmt{
		Results: results,
	}
}

// TakeArgs takes the arguments from the method
func TakeArgs(name string, method string, declare bool) ast.Stmt {
	var tk = token.DEFINE
	if !declare {
		tk = token.ASSIGN
	}

	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			NewIdentObj(name),
		},
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   NewIdentObjVar(varBeyondContext),
					Sel: NewIdentObj(method),
				},
			},
		},
		Tok: tk,
	}
}

// SetArgs set arguments
func SetArgs(method string, name string) ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   NewIdentObjVar(varBeyondContext),
				Sel: NewIdentObj(method),
			},
			Args: []ast.Expr{
				NewIdentObj(name),
			},
		},
	}
}
