package web_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/web"
)

func TestNewTemplateRenderer_InvalidPath(t *testing.T) {
	// Test with non-existent directory
	_, err := web.NewTemplateRenderer("/nonexistent/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no files")
}

func TestNewTemplateRenderer_WithValidTemplates(t *testing.T) {
	// Create temporary directory with test templates
	tempDir := t.TempDir()

	// Create directory structure
	layoutsDir := filepath.Join(tempDir, "layouts")
	pagesDir := filepath.Join(tempDir, "pages")
	componentsDir := filepath.Join(tempDir, "components")

	err := os.MkdirAll(layoutsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(pagesDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// Create test template files
	layoutTemplate := `{{define "base"}}<!DOCTYPE html><html><head><title>{{.Title}}</title></head><body>{{template "content" .}}</body></html>{{end}}`
	err = os.WriteFile(filepath.Join(layoutsDir, "base.html"), []byte(layoutTemplate), 0644)
	require.NoError(t, err)

	pageTemplate := `{{define "test-page"}}{{template "base" .}}{{end}}{{define "content"}}<h1>{{.Title}}</h1><p>{{.Message}}</p>{{end}}`
	err = os.WriteFile(filepath.Join(pagesDir, "test-page.html"), []byte(pageTemplate), 0644)
	require.NoError(t, err)

	componentTemplate := `{{define "nav"}}<nav><a href="/">Home</a></nav>{{end}}`
	err = os.WriteFile(filepath.Join(componentsDir, "nav.html"), []byte(componentTemplate), 0644)
	require.NoError(t, err)

	// Test renderer creation
	renderer, err := web.NewTemplateRenderer(tempDir)
	require.NoError(t, err)
	assert.NotNil(t, renderer)

	// Test implementing echo.Renderer interface
	var echoRenderer echo.Renderer = renderer
	assert.NotNil(t, echoRenderer)
}

func TestTemplateRenderer_Render(t *testing.T) {
	// Create temporary directory with test templates
	tempDir := t.TempDir()

	// Create directory structure
	layoutsDir := filepath.Join(tempDir, "layouts")
	pagesDir := filepath.Join(tempDir, "pages")
	componentsDir := filepath.Join(tempDir, "components")

	err := os.MkdirAll(layoutsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(pagesDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// Create minimal layout template (required by the template renderer)
	layoutTemplate := `{{define "base"}}<!DOCTYPE html><html><body>{{template "content" .}}</body></html>{{end}}`
	err = os.WriteFile(filepath.Join(layoutsDir, "base.html"), []byte(layoutTemplate), 0644)
	require.NoError(t, err)

	// Create minimal component template (required by the template renderer)
	componentTemplate := `{{define "nav"}}<nav>Navigation</nav>{{end}}`
	err = os.WriteFile(filepath.Join(componentsDir, "nav.html"), []byte(componentTemplate), 0644)
	require.NoError(t, err)

	// Create simple test template
	testTemplate := `{{define "simple"}}Hello {{.Name}}! You have {{.Count}} items.{{end}}`
	err = os.WriteFile(filepath.Join(pagesDir, "simple.html"), []byte(testTemplate), 0644)
	require.NoError(t, err)

	// Create renderer
	renderer, err := web.NewTemplateRenderer(tempDir)
	require.NoError(t, err)
	assert.NotNil(t, renderer)

	// Setup Echo context for testing
	e := echo.New()
	e.Renderer = renderer

	// Test data
	data := map[string]any{
		"Name":  "Test User",
		"Count": 42,
	}

	// Create a buffer to capture output
	var output strings.Builder

	// Test rendering
	err = renderer.Render(&output, "simple", data, nil)
	require.NoError(t, err)

	// Verify output
	result := output.String()
	assert.Contains(t, result, "Hello Test User!")
	assert.Contains(t, result, "You have 42 items.")
}

func TestTemplateRenderer_RenderWithHelperFunctions(t *testing.T) {
	// Create temporary directory with test templates
	tempDir := t.TempDir()

	// Create directory structure
	layoutsDir := filepath.Join(tempDir, "layouts")
	pagesDir := filepath.Join(tempDir, "pages")
	componentsDir := filepath.Join(tempDir, "components")

	err := os.MkdirAll(layoutsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(pagesDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// Create minimal layout template (required by the template renderer)
	layoutTemplate := `{{define "base"}}<!DOCTYPE html><html><body>{{template "content" .}}</body></html>{{end}}`
	err = os.WriteFile(filepath.Join(layoutsDir, "base.html"), []byte(layoutTemplate), 0644)
	require.NoError(t, err)

	// Create minimal component template (required by the template renderer)
	componentTemplate := `{{define "nav"}}<nav>Navigation</nav>{{end}}`
	err = os.WriteFile(filepath.Join(componentsDir, "nav.html"), []byte(componentTemplate), 0644)
	require.NoError(t, err)

	// Create template that uses helper functions
	testTemplate := `{{define "math"}}{{add 1 2}} + {{sub 10 3}} = {{add (add 1 2) (sub 10 3)}}{{end}}`
	err = os.WriteFile(filepath.Join(pagesDir, "math.html"), []byte(testTemplate), 0644)
	require.NoError(t, err)

	// Create renderer
	renderer, err := web.NewTemplateRenderer(tempDir)
	require.NoError(t, err)

	// Create a buffer to capture output
	var output strings.Builder

	// Test rendering with helper functions
	err = renderer.Render(&output, "math", nil, nil)
	require.NoError(t, err)

	// Verify output
	result := output.String()
	assert.Contains(t, result, "3 + 7 = 10")
}

func TestTemplateRenderer_RenderInvalidTemplate(t *testing.T) {
	// Create temporary directory with test templates
	tempDir := t.TempDir()

	// Create directory structure
	layoutsDir := filepath.Join(tempDir, "layouts")
	pagesDir := filepath.Join(tempDir, "pages")
	componentsDir := filepath.Join(tempDir, "components")

	err := os.MkdirAll(layoutsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(pagesDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// Create minimal required templates
	layoutTemplate := `{{define "base"}}<!DOCTYPE html><html><body>{{template "content" .}}</body></html>{{end}}`
	err = os.WriteFile(filepath.Join(layoutsDir, "base.html"), []byte(layoutTemplate), 0644)
	require.NoError(t, err)

	componentTemplate := `{{define "nav"}}<nav>Navigation</nav>{{end}}`
	err = os.WriteFile(filepath.Join(componentsDir, "nav.html"), []byte(componentTemplate), 0644)
	require.NoError(t, err)

	// Create a dummy template file
	testTemplate := `{{define "dummy"}}Dummy{{end}}`
	err = os.WriteFile(filepath.Join(pagesDir, "dummy.html"), []byte(testTemplate), 0644)
	require.NoError(t, err)

	// Create renderer
	renderer, err := web.NewTemplateRenderer(tempDir)
	require.NoError(t, err)

	// Create a buffer to capture output
	var output strings.Builder

	// Test rendering non-existent template
	err = renderer.Render(&output, "nonexistent", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestTemplateRenderer_EmptyDirectories(t *testing.T) {
	// Create temporary directory with empty subdirectories
	tempDir := t.TempDir()

	// Create empty directory structure
	layoutsDir := filepath.Join(tempDir, "layouts")
	pagesDir := filepath.Join(tempDir, "pages")
	componentsDir := filepath.Join(tempDir, "components")

	err := os.MkdirAll(layoutsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(pagesDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// This should fail because no template files exist
	_, err = web.NewTemplateRenderer(tempDir)
	require.Error(t, err)
}

func TestTemplateRenderer_RecursiveTemplateLoading(t *testing.T) {
	// Create temporary directory with nested template structure
	tempDir := t.TempDir()

	// Create directory structure
	layoutsDir := filepath.Join(tempDir, "layouts")
	pagesDir := filepath.Join(tempDir, "pages")
	componentsDir := filepath.Join(tempDir, "components")

	// Create nested directories in pages
	transactionsDir := filepath.Join(pagesDir, "transactions")
	reportsDir := filepath.Join(pagesDir, "reports")

	err := os.MkdirAll(layoutsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(pagesDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(transactionsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(reportsDir, 0755)
	require.NoError(t, err)

	// Create minimal required templates
	layoutTemplate := `{{define "base"}}<!DOCTYPE html><html><body>{{template "content" .}}</body></html>{{end}}`
	err = os.WriteFile(filepath.Join(layoutsDir, "base.html"), []byte(layoutTemplate), 0644)
	require.NoError(t, err)

	componentTemplate := `{{define "nav"}}<nav>Navigation</nav>{{end}}`
	err = os.WriteFile(filepath.Join(componentsDir, "nav.html"), []byte(componentTemplate), 0644)
	require.NoError(t, err)

	// Create root page template
	rootPageTemplate := `{{define "home"}}Welcome to the home page{{end}}`
	err = os.WriteFile(filepath.Join(pagesDir, "home.html"), []byte(rootPageTemplate), 0644)
	require.NoError(t, err)

	// Create nested page templates
	transactionIndexTemplate := `{{define "pages/transactions/index"}}Transaction index page{{end}}`
	err = os.WriteFile(filepath.Join(transactionsDir, "index.html"), []byte(transactionIndexTemplate), 0644)
	require.NoError(t, err)

	transactionNewTemplate := `{{define "pages/transactions/new"}}New transaction page{{end}}`
	err = os.WriteFile(filepath.Join(transactionsDir, "new.html"), []byte(transactionNewTemplate), 0644)
	require.NoError(t, err)

	reportsIndexTemplate := `{{define "pages/reports/index"}}Reports index page{{end}}`
	err = os.WriteFile(filepath.Join(reportsDir, "index.html"), []byte(reportsIndexTemplate), 0644)
	require.NoError(t, err)

	// Create renderer
	renderer, err := web.NewTemplateRenderer(tempDir)
	require.NoError(t, err)
	assert.NotNil(t, renderer)

	// Test rendering root template
	var output strings.Builder
	err = renderer.Render(&output, "home", nil, nil)
	require.NoError(t, err)
	assert.Contains(t, output.String(), "Welcome to the home page")

	// Test rendering nested templates
	output.Reset()
	err = renderer.Render(&output, "pages/transactions/index", nil, nil)
	require.NoError(t, err)
	assert.Contains(t, output.String(), "Transaction index page")

	output.Reset()
	err = renderer.Render(&output, "pages/transactions/new", nil, nil)
	require.NoError(t, err)
	assert.Contains(t, output.String(), "New transaction page")

	output.Reset()
	err = renderer.Render(&output, "pages/reports/index", nil, nil)
	require.NoError(t, err)
	assert.Contains(t, output.String(), "Reports index page")
}

func TestTemplateRenderer_DerefFunction(t *testing.T) {
	// Create temporary directory with test templates
	tempDir := t.TempDir()

	// Create directory structure
	layoutsDir := filepath.Join(tempDir, "layouts")
	pagesDir := filepath.Join(tempDir, "pages")
	componentsDir := filepath.Join(tempDir, "components")

	err := os.MkdirAll(layoutsDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(pagesDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(componentsDir, 0755)
	require.NoError(t, err)

	// Create minimal required templates
	layoutTemplate := `{{define "base"}}<!DOCTYPE html><html><body>{{template "content" .}}</body></html>{{end}}`
	err = os.WriteFile(filepath.Join(layoutsDir, "base.html"), []byte(layoutTemplate), 0644)
	require.NoError(t, err)

	componentTemplate := `{{define "nav"}}<nav>Navigation</nav>{{end}}`
	err = os.WriteFile(filepath.Join(componentsDir, "nav.html"), []byte(componentTemplate), 0644)
	require.NoError(t, err)

	// Create template that uses deref function
	testTemplate := `{{define "deref-test"}}Active: {{deref .IsActive}} | Enabled: {{deref .IsEnabled}}{{end}}`
	err = os.WriteFile(filepath.Join(pagesDir, "deref-test.html"), []byte(testTemplate), 0644)
	require.NoError(t, err)

	// Create renderer
	renderer, err := web.NewTemplateRenderer(tempDir)
	require.NoError(t, err)

	// Test data with boolean pointers
	trueVal := true
	falseVal := false
	data := map[string]any{
		"IsActive":  &trueVal,
		"IsEnabled": &falseVal,
	}

	// Create a buffer to capture output
	var output strings.Builder

	// Test rendering with deref function
	err = renderer.Render(&output, "deref-test", data, nil)
	require.NoError(t, err)

	// Verify output
	result := output.String()
	assert.Contains(t, result, "Active: true")
	assert.Contains(t, result, "Enabled: false")

	// Test with nil pointers
	output.Reset()
	dataNil := map[string]any{
		"IsActive":  (*bool)(nil),
		"IsEnabled": (*bool)(nil),
	}

	err = renderer.Render(&output, "deref-test", dataNil, nil)
	require.NoError(t, err)

	// Verify output with nil values (should render as false)
	result = output.String()
	assert.Contains(t, result, "Active: false")
	assert.Contains(t, result, "Enabled: false")
}
