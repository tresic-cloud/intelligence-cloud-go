package generated_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// operationIDs returns every operationId declared in the pinned OpenAPI spec
// by scanning the YAML for "operationId:" lines. This avoids a YAML parser
// dependency while still being robust: the OpenAPI spec requires operationId
// to be a scalar string value on its own line.
func operationIDs(t *testing.T) []string {
	t.Helper()

	specPath := specFilePath(t)
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("cannot read pinned OpenAPI spec at %s: %v", specPath, err)
	}
	if len(data) == 0 {
		t.Fatalf("pinned OpenAPI spec at %s is empty", specPath)
	}

	var ids []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "operationId:") {
			id := strings.TrimSpace(strings.TrimPrefix(trimmed, "operationId:"))
			id = strings.Trim(id, `"'`)
			if id != "" {
				ids = append(ids, id)
			}
		}
	}

	if len(ids) == 0 {
		t.Fatalf("no operationIds found in %s", specPath)
	}
	return ids
}

// exportedFuncNames returns all exported function and method names declared
// in the given Go source file.
func exportedFuncNames(t *testing.T, filename string) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("cannot parse %s: %v", filename, err)
	}

	names := make(map[string]bool)
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.IsExported() {
			names[fn.Name.Name] = true
		}
	}
	return names
}

// exportedTypeNames returns all exported type names declared in the given
// Go source file.
func exportedTypeNames(t *testing.T, filename string) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("cannot parse %s: %v", filename, err)
	}

	names := make(map[string]bool)
	for _, decl := range node.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if ts.Name.IsExported() {
				names[ts.Name.Name] = true
			}
		}
	}
	return names
}

// specFilePath returns the absolute path to testdata/openapi.yaml relative
// to this test file's location.
func specFilePath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "openapi.yaml")
}

// clientGenPath returns the absolute path to client.gen.go relative to this
// test file's location.
func clientGenPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "client.gen.go")
}

// typesGenPath returns the absolute path to types.gen.go relative to this
// test file's location.
func typesGenPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "types.gen.go")
}

// opIDToFuncName converts an OpenAPI operationId to the expected exported
// Go function name that oapi-codegen generates. The convention is to
// upper-case the first letter (PascalCase). Since the canonical spec already
// uses camelCase operationIds, this simply uppercases the first character.
func opIDToFuncName(opID string) string {
	if opID == "" {
		return ""
	}
	return strings.ToUpper(opID[:1]) + opID[1:]
}

// TestDrift_AllOperationIDsCovered verifies that every operationId in the
// pinned OpenAPI spec has a corresponding exported function (client method)
// in client.gen.go. If this test fails, it means either:
//   - A new operation was added to the spec but codegen was not re-run.
//   - An operationId was renamed in the spec but codegen was not re-run.
//   - The generated client.gen.go was hand-edited and a function was removed.
func TestDrift_AllOperationIDsCovered(t *testing.T) {
	ids := operationIDs(t)
	t.Logf("Found %d operationIds in pinned spec", len(ids))

	clientPath := clientGenPath(t)
	funcNames := exportedFuncNames(t, clientPath)

	// We check for both the direct method name (e.g. Login, GetMe) and the
	// request constructor (e.g. NewLoginRequest, NewGetMeRequest).
	var missing []string
	for _, id := range ids {
		funcName := opIDToFuncName(id)
		constructorName := "New" + funcName + "Request"

		hasMethod := funcNames[funcName]
		hasConstructor := funcNames[constructorName]

		if !hasMethod && !hasConstructor {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 {
		t.Errorf("operationIds without generated functions (%d/%d):\n  %s",
			len(missing), len(ids), strings.Join(missing, "\n  "))
	}
}

// TestDrift_TypesGenHasExportedTypes verifies that types.gen.go contains at
// least one exported type symbol. This is a sanity check that the types file
// was generated and is not empty.
func TestDrift_TypesGenHasExportedTypes(t *testing.T) {
	typesPath := typesGenPath(t)
	names := exportedTypeNames(t, typesPath)
	if len(names) == 0 {
		t.Fatal("types.gen.go contains no exported types; codegen may not have been run")
	}
	t.Logf("types.gen.go exports %d types", len(names))
}

// TestDrift_ClientGenFileExists verifies client.gen.go exists and is non-empty.
func TestDrift_ClientGenFileExists(t *testing.T) {
	path := clientGenPath(t)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("client.gen.go not found at %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("client.gen.go is empty at %s", path)
	}
}

// TestDrift_TypesGenFileExists verifies types.gen.go exists and is non-empty.
func TestDrift_TypesGenFileExists(t *testing.T) {
	path := typesGenPath(t)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("types.gen.go not found at %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("types.gen.go is empty at %s", path)
	}
}

// TestDrift_SpecCommitFileExists verifies testdata/openapi.commit is present.
func TestDrift_SpecCommitFileExists(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	commitPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "openapi.commit")
	data, err := os.ReadFile(commitPath)
	if err != nil {
		t.Fatalf("testdata/openapi.commit not found: %v", err)
	}
	sha := strings.TrimSpace(string(data))
	if len(sha) < 40 {
		t.Fatalf("testdata/openapi.commit does not contain a valid SHA (got %q)", sha)
	}
	t.Logf("Pinned spec commit: %s", sha)
}
