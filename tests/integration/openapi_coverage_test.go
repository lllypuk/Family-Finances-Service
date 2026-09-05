package integration_test

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"family-budget-service/internal/testhelpers"
)

const (
	errorSchemaRef     = "#/components/schemas/Error"
	responsesRefPrefix = "#/components/responses/"
	jsonMediaType      = "application/json"
)

// openAPISpec — минимальная модель спецификации: только то, что проверяет тест.
type openAPISpec struct {
	Paths      map[string]openAPIPathItem `yaml:"paths"`
	Components struct {
		Responses map[string]openAPIResponse `yaml:"responses"`
	} `yaml:"components"`
}

type openAPIPathItem struct {
	Get    *openAPIOperation `yaml:"get"`
	Put    *openAPIOperation `yaml:"put"`
	Post   *openAPIOperation `yaml:"post"`
	Patch  *openAPIOperation `yaml:"patch"`
	Delete *openAPIOperation `yaml:"delete"`
}

func (p openAPIPathItem) operations() map[string]*openAPIOperation {
	byMethod := make(map[string]*openAPIOperation)
	for method, op := range map[string]*openAPIOperation{
		http.MethodGet:    p.Get,
		http.MethodPut:    p.Put,
		http.MethodPost:   p.Post,
		http.MethodPatch:  p.Patch,
		http.MethodDelete: p.Delete,
	} {
		if op != nil {
			byMethod[method] = op
		}
	}

	return byMethod
}

type openAPIOperation struct {
	OperationID string                     `yaml:"operationId"`
	Responses   map[string]openAPIResponse `yaml:"responses"`
}

type openAPIResponse struct {
	Ref     string                      `yaml:"$ref"`
	Content map[string]openAPIMediaType `yaml:"content"`
}

type openAPIMediaType struct {
	Schema struct {
		Ref string `yaml:"$ref"`
	} `yaml:"schema"`
}

func loadOpenAPISpec(t *testing.T) openAPISpec {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(testhelpers.RepoRoot(t), "docs", "api", "openapi.yaml"))
	require.NoError(t, err, "docs/api/openapi.yaml не читается")

	var spec openAPISpec
	require.NoError(t, yaml.Unmarshal(raw, &spec), "docs/api/openapi.yaml не парсится")
	require.NotEmpty(t, spec.Paths)

	return spec
}

// specRoutes переводит пути спецификации в форму echo: {id} → :id.
func specRoutes(spec openAPISpec) map[string]bool {
	pathParam := regexp.MustCompile(`\{([^}]+)\}`)
	routes := make(map[string]bool)
	for path, item := range spec.Paths {
		echoPath := pathParam.ReplaceAllString(path, ":$1")
		for method := range item.operations() {
			routes[method+" "+echoPath] = true
		}
	}

	return routes
}

// registeredAPIRoutes — роуты /api/v1 плюс GET /health. Заглушки echo для групп
// (Any("") и Any("/*") из Group.Use) регистрируются с методом echo.RouteNotFound.
func registeredAPIRoutes(t *testing.T) []string {
	t.Helper()

	ts := testhelpers.SetupHTTPServer(t)
	var routes []string
	for _, route := range ts.Server.Echo().Routes() {
		if route.Method == echo.RouteNotFound {
			continue
		}
		if route.Path != "/health" && !strings.HasPrefix(route.Path, "/api/v1") {
			continue
		}
		routes = append(routes, route.Method+" "+route.Path)
	}
	require.NotEmpty(t, routes)
	sort.Strings(routes)

	return routes
}

func hasErrorResponse(spec openAPISpec, op *openAPIOperation) bool {
	for code, resp := range op.Responses {
		if !strings.HasPrefix(code, "4") {
			continue
		}
		if name, ok := strings.CutPrefix(resp.Ref, responsesRefPrefix); ok {
			resp = spec.Components.Responses[name]
		}
		if resp.Content[jsonMediaType].Schema.Ref == errorSchemaRef {
			return true
		}
	}

	return false
}

func TestOpenAPISpec_CoversRegisteredRoutes(t *testing.T) {
	spec := loadOpenAPISpec(t)
	described := specRoutes(spec)

	var missing []string
	for _, route := range registeredAPIRoutes(t) {
		if !described[route] {
			missing = append(missing, route)
		}
	}

	assert.Emptyf(t, missing, "роуты без описания в docs/api/openapi.yaml:\n  %s", strings.Join(missing, "\n  "))
}

func TestOpenAPISpec_OperationsHaveIDAndErrorResponse(t *testing.T) {
	spec := loadOpenAPISpec(t)

	seen := make(map[string]string)
	for path, item := range spec.Paths {
		for method, op := range item.operations() {
			route := method + " " + path
			assert.NotEmptyf(t, op.OperationID, "%s: нет operationId", route)
			assert.Truef(t, hasErrorResponse(spec, op), "%s: нет ответа 4xx со схемой Error", route)
			if op.OperationID != "" {
				assert.Emptyf(t, seen[op.OperationID], "%s: operationId %q уже занят %s",
					route, op.OperationID, seen[op.OperationID])
				seen[op.OperationID] = route
			}
		}
	}
}
