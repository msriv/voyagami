package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/msriv/voyagami/services/api/internal/server"
)

const specPath = "../../../../api/openapi.yaml"

func loadSpec(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile(specPath)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	return doc
}

func TestSpecIsValidOpenAPI31(t *testing.T) {
	doc := loadSpec(t)
	if !strings.HasPrefix(doc.OpenAPI, "3.1.") {
		t.Fatalf("openapi = %q, want 3.1.x", doc.OpenAPI)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("spec is not valid: %v", err)
	}
}

// Every operation in the spec must be served by the router. A path that
// answers 404 or 405 means the spec and the server have drifted apart.
func TestEverySpecOperationIsRouted(t *testing.T) {
	doc := loadSpec(t)
	h := server.NewHandler(nilLogger())

	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			t.Run(method+" "+path, func(t *testing.T) {
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
				if rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed {
					t.Errorf("status = %d, route is not served", rec.Code)
				}
			})
		}
	}
}
