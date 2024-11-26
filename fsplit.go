package fsplit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/imports"
)

// RunFsplit runs the fsplit tool
// It extracts functions from the package, creates single function files,
// and removes functions from the original files
func RunFsplit(packagePath string) error {
	funcFiles, err := extractFunctions(packagePath)
	if err != nil {
		return fmt.Errorf("Error detecting and extracting functions: %v", err)
	}

	if err := createSingleFunctionFiles(funcFiles); err != nil {
		return fmt.Errorf("Error creating single function files: %v", err)
	}

	if err = removeFunctions(packagePath); err != nil {
		return fmt.Errorf("Error removing functions: %v", err)
	}

	return nil
}

// SingleFunctionFile represents a single function file
type SingleFunctionFile struct {
	// FileName is the name of the single function file
	FileName string
	// Package is the package declaration of the file
	Package string
	// Imports is the import declarations of the file
	Imports string
	// Func is the function declaration of the file
	Func string
}

// isNotTarget checks if the file matches one of the following criteria:
// 1. It is a test file
// 2. It is a generated file
// 3. It contains less or equal to 1 function
func isNotTarget(file *ast.File) bool {
	// Check if the file is a test file by its name
	if len(file.Name.Name) > 4 && file.Name.Name[len(file.Name.Name)-4:] == "_test" {
		return true
	}

	// Check if the file is a generated file
	for _, comment := range file.Comments {
		if strings.Contains(comment.Text(), "Code generated") {
			return true
		}
	}

	// Check if the file contains less or equal to 1 function
	funcCount := 0
	for _, decl := range file.Decls {
		if _, ok := decl.(*ast.FuncDecl); ok {
			funcCount++
		}
	}
	return funcCount <= 1
}

// newFileName generates a new file name for the single function file
func newFileName(original string, recv string, funcName string) string {
	// Remove .go extension
	stem := original[:len(original)-3]
	if strings.HasSuffix(stem, ".fsplit.go") {
		// get the original stem
		split := strings.Split(original, ".")
		stem = strings.Join(split[:len(split)-4], ".")
	}
	if recv == "" {
		recv = "_"
	}
	return stem + "." + recv + "." + funcName + ".fsplit.go"
}

// getRecvTypeName gets the receiver type name of the function if it exists
// If the function does not have a receiver, it returns an empty string
func getRecvTypeName(decl *ast.FuncDecl) string {
	if decl.Recv == nil {
		return ""
	}
	recv := decl.Recv.List[0]
	switch recvType := recv.Type.(type) {
	case *ast.StarExpr:
		return recvType.X.(*ast.Ident).Name
	case *ast.Ident:
		return recvType.Name
	}
	return ""
}

// extractFunctions extracts functions from the package and returns a list of SingleFunctionFile
func extractFunctions(packagePath string) ([]SingleFunctionFile, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packagePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var funcFiles []SingleFunctionFile
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			if isNotTarget(file) {
				continue
			}

			// init function can be declared multiple times
			initCnt := 0

			// Extract package declaration from the file.
			// This is needed to copy comments before the package declaration.
			var buf bytes.Buffer
			err := printer.Fprint(&buf, fset, file)
			if err != nil {
				return nil, err
			}
			fileContent := buf.String()
			packageDecl := fileContent[:fset.Position(file.Decls[0].Pos()).Offset]

			imports := ""
			for _, decl := range file.Decls {
				switch decl := decl.(type) {
				case *ast.GenDecl:
					if decl.Tok == token.IMPORT {
						imports += fileContent[fset.Position(decl.Pos()).Offset:fset.Position(decl.End()).Offset] + "\n"
					}
				case *ast.FuncDecl:
					var funcBuf bytes.Buffer
					err := printer.Fprint(&funcBuf, fset, &printer.CommentedNode{Node: decl, Comments: file.Comments})
					if err != nil {
						return nil, err
					}
					recvTypeName := getRecvTypeName(decl)
					funcName := decl.Name.Name
					if funcName == "init" {
						initCnt++
						funcName = fmt.Sprintf("init-%03d", initCnt)
					}
					newFileName := newFileName(fset.Position(file.Name.Pos()).Filename, recvTypeName, funcName)
					funcFiles = append(funcFiles, SingleFunctionFile{
						FileName: newFileName,
						Package:  packageDecl,
						Imports:  imports,
						Func:     funcBuf.String(),
					})
				}
			}
		}
	}

	return funcFiles, nil
}

// createSingleFunctionFiles creates single function files from the list of SingleFunctionFile
func createSingleFunctionFiles(funcFiles []SingleFunctionFile) error {
	for _, funcFile := range funcFiles {
		fileContent := funcFile.Package + funcFile.Imports + funcFile.Func
		formatted, err := imports.Process(funcFile.FileName, []byte(fileContent), nil)
		if err != nil {
			return err
		}
		err = os.WriteFile(funcFile.FileName, formatted, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// isCommentAssociatedWithFunction checks if the comment is associated with any function
func isCommentAssociatedWithFunction(comment *ast.CommentGroup, file *ast.File) bool {
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			// Check if the comment is the function's doc comment
			if funcDecl.Doc == comment {
				return true
			}

			// Check if the comment is inside the function
			if funcDecl.Pos() < comment.Pos() && comment.Pos() < funcDecl.End() {
				return true
			}
		}
	}

	return false
}

// removeUnnecessaryComments removes unnecessary comments from the file
// Unnecessary comments are comments that are associated with any function
func removeUnnecessaryComments(file *ast.File) {
	var comments []*ast.CommentGroup
	for _, comment := range file.Comments {
		if !isCommentAssociatedWithFunction(comment, file) {
			comments = append(comments, comment)
		}
	}
	file.Comments = comments
}

// removeFunctionsFromFile removes functions from the file
// This should be called after removeUnnecessaryComments
func removeFunctionsFromFile(file *ast.File) {
	var decls []ast.Decl
	for _, decl := range file.Decls {
		if _, ok := decl.(*ast.FuncDecl); !ok {
			decls = append(decls, decl)
		}
	}
	file.Decls = decls
}

// removeFunctions removes functions from the package
func removeFunctions(packagePath string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packagePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			if isNotTarget(file) {
				continue
			}

			removeUnnecessaryComments(file)
			removeFunctionsFromFile(file)

			var buf bytes.Buffer
			err := printer.Fprint(&buf, fset, file)
			if err != nil {
				return err
			}

			// Remove unused imports
			formatted, err := imports.Process(fileName, buf.Bytes(), nil)

			err = os.WriteFile(fileName, formatted, 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
