package scanner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestAnalyzeResponseStruct(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		responseType   string
		wantHeaders    int
		wantStatusCode bool
		wantError      bool
	}{
		{
			name: "struct with header tags",
			code: `package test
type Response struct {
	Data     string ` + "`json:\"data\"`" + `
	Location string ` + "`header:\"Location\"`" + `
	ETag     string ` + "`header:\"ETag,omitempty\"`" + `
}`,
			responseType:   "Response",
			wantHeaders:    2,
			wantStatusCode: false,
		},
		{
			name: "struct with status code",
			code: `package test
type Response struct {
	Data   string ` + "`json:\"data\"`" + `
	Status int    ` + "`response:\"httpstatus\"`" + `
}`,
			responseType:   "Response",
			wantHeaders:    0,
			wantStatusCode: true,
		},
		{
			name: "struct with both",
			code: `package test
type Response struct {
	Data     string ` + "`json:\"data\"`" + `
	Location string ` + "`header:\"Location\"`" + `
	Status   int    ` + "`response:\"httpstatus\"`" + `
}`,
			responseType:   "Response",
			wantHeaders:    1,
			wantStatusCode: true,
		},
		{
			name: "struct without metadata",
			code: `package test
type Response struct {
	Data string ` + "`json:\"data\"`" + `
	Name string ` + "`json:\"name\"`" + `
}`,
			responseType:   "Response",
			wantHeaders:    0,
			wantStatusCode: false,
		},
		{
			name: "invalid status field type",
			code: `package test
type Response struct {
	Data   string ` + "`json:\"data\"`" + `
	Status string ` + "`response:\"httpstatus\"`" + `
}`,
			responseType: "Response",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse code
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse code: %v", err)
			}

			// Create scanner
			s := &Scanner{
				currentPackageName: "test",
				typeSpecs:          make(map[string]*ast.TypeSpec),
			}

			// Collect type specs
			for _, decl := range f.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							s.typeSpecs[typeSpec.Name.Name] = typeSpec
						}
					}
				}
			}

			// Create response type info
			responseType := &TypeInfo{
				Name:        tt.responseType,
				PackageName: "",
			}

			// Analyze response struct
			metadata, err := s.analyzeResponseStruct(responseType)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check if metadata should be nil (no tags)
			if tt.wantHeaders == 0 && !tt.wantStatusCode {
				if metadata != nil {
					t.Errorf("expected nil metadata, got %+v", metadata)
				}
				return
			}

			if metadata == nil {
				t.Fatal("expected metadata, got nil")
			}

			// Check header fields
			if len(metadata.HeaderFields) != tt.wantHeaders {
				t.Errorf("expected %d header fields, got %d", tt.wantHeaders, len(metadata.HeaderFields))
			}

			// Check status code field
			hasStatusCode := metadata.StatusCodeField != ""
			if hasStatusCode != tt.wantStatusCode {
				t.Errorf("expected status code field: %v, got: %v", tt.wantStatusCode, hasStatusCode)
			}

			// Additional checks for specific test cases
			if tt.name == "struct with header tags" {
				// Check Location header
				found := false
				for _, hf := range metadata.HeaderFields {
					if hf.HeaderName == "Location" && !hf.OmitEmpty {
						found = true
					}
				}
				if !found {
					t.Error("expected Location header without omitempty")
				}

				// Check ETag header with omitempty
				found = false
				for _, hf := range metadata.HeaderFields {
					if hf.HeaderName == "ETag" && hf.OmitEmpty {
						found = true
					}
				}
				if !found {
					t.Error("expected ETag header with omitempty")
				}
			}

			if tt.name == "struct with status code" {
				if metadata.StatusCodeField != "Status" {
					t.Errorf("expected status code field 'Status', got '%s'", metadata.StatusCodeField)
				}
			}
		})
	}
}

func TestAnalyzeResponseStructNilAndPrimitives(t *testing.T) {
	s := &Scanner{
		currentPackageName: "test",
		typeSpecs:          make(map[string]*ast.TypeSpec),
	}

	// Nil response type
	metadata, err := s.analyzeResponseStruct(nil)
	if err != nil {
		t.Errorf("expected no error for nil type, got: %v", err)
	}
	if metadata != nil {
		t.Errorf("expected nil metadata for nil type, got: %+v", metadata)
	}

	// Primitive response type
	primitiveType := &TypeInfo{
		Name:        "string",
		IsPrimitive: true,
	}
	metadata, err = s.analyzeResponseStruct(primitiveType)
	if err != nil {
		t.Errorf("expected no error for primitive type, got: %v", err)
	}
	if metadata != nil {
		t.Errorf("expected nil metadata for primitive type, got: %+v", metadata)
	}

	// Slice response type
	sliceType := &TypeInfo{
		Name:    "Post",
		IsSlice: true,
	}
	metadata, err = s.analyzeResponseStruct(sliceType)
	if err != nil {
		t.Errorf("expected no error for slice type, got: %v", err)
	}
	if metadata != nil {
		t.Errorf("expected nil metadata for slice type, got: %+v", metadata)
	}

	// External package type
	externalType := &TypeInfo{
		Name:        "User",
		PackageName: "models",
	}
	metadata, err = s.analyzeResponseStruct(externalType)
	if err != nil {
		t.Errorf("expected no error for external type, got: %v", err)
	}
	if metadata != nil {
		t.Errorf("expected nil metadata for external type, got: %+v", metadata)
	}
}
