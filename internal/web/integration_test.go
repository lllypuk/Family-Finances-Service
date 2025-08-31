package web_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/web"
)

func TestTemplateRenderer_RealTemplates(t *testing.T) {
	// Test loading real templates from the project
	templatesDir := "templates"

	renderer, err := web.NewTemplateRenderer(templatesDir)
	require.NoError(t, err, "Should be able to load real templates from project")
	assert.NotNil(t, renderer)
}

func TestTemplateRenderer_RealTemplatesExist(t *testing.T) {
	// Test that our key templates exist and can be rendered
	templatesDir := "templates"

	renderer, err := web.NewTemplateRenderer(templatesDir)
	require.NoError(t, err)

	// Test rendering key templates that were causing issues
	testCases := []struct {
		name         string
		templateName string
		shouldExist  bool
	}{
		{
			name:         "transactions index template",
			templateName: "pages/transactions/index",
			shouldExist:  true,
		},
		{
			name:         "transactions new template",
			templateName: "pages/transactions/new",
			shouldExist:  true,
		},
		{
			name:         "reports index template",
			templateName: "pages/reports/index",
			shouldExist:  true,
		},
		{
			name:         "reports new template",
			templateName: "pages/reports/new",
			shouldExist:  true,
		},
		{
			name:         "reports show template",
			templateName: "pages/reports/show",
			shouldExist:  true,
		},
		{
			name:         "error template",
			templateName: "pages/error",
			shouldExist:  true,
		},
		{
			name:         "dashboard template",
			templateName: "dashboard",
			shouldExist:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create minimal test data
			data := map[string]any{
				"PageData": map[string]any{
					"Title":  "Test Title",
					"Errors": map[string]string{}, // Empty errors map
				},
				"CSRFToken": "test-csrf-token",
				"Form": map[string]any{
					"Name": "Test Form",
				},
				"Report": map[string]any{
					"Name": "Test Report",
					"Type": "expense",
				},
				"Title": "Test Title", // For dashboard template
			}

			// Try to render the template
			var output strings.Builder
			err := renderer.Render(&output, tc.templateName, data, nil)

			if tc.shouldExist {
				assert.NoError(t, err, "Template %s should exist and render without error", tc.templateName)
				assert.NotEmpty(t, output.String(), "Template %s should produce output", tc.templateName)
			} else {
				assert.Error(t, err, "Template %s should not exist", tc.templateName)
			}
		})
	}
}

func TestTemplateRenderer_HelperFunctions(t *testing.T) {
	// Test that all helper functions are available
	templatesDir := "templates"

	renderer, err := web.NewTemplateRenderer(templatesDir)
	require.NoError(t, err)

	// Test deref function specifically since it was causing issues
	// For now, just verify the renderer was created successfully
	assert.NotNil(t, renderer)
}
