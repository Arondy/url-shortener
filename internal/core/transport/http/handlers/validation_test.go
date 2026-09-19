package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Arondy/url-shortener/internal/core/transport/http/handlers"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type validationFixture struct {
	Required string `json:"required_field" validate:"required"`
	Min      string `json:"min_field" validate:"min=4"`
	Max      string `json:"max_field" validate:"max=5"`
	URL      string `json:"url_field" validate:"url"`
	Other    string `json:"other_field" validate:"oneof=red green"`
}

func collectErrors(t *testing.T, v any) validator.ValidationErrors {
	t.Helper()
	err := handlers.Validate.Struct(v)
	require.Error(t, err)
	valErrs, ok := err.(validator.ValidationErrors)
	require.True(t, ok, "expected validator.ValidationErrors, got %T", err)
	return valErrs
}

func TestFormatValidation_MapsAllTags(t *testing.T) {
	t.Parallel()
	valErrs := collectErrors(t, validationFixture{
		Min:   "abc",
		Max:   "toolong",
		URL:   "not-a-url",
		Other: "blue",
	})

	got := handlers.FormatValidation(valErrs)

	assert.Equal(t, "Field is required", got["required_field"])
	assert.Equal(t, "Field must be at least 4", got["min_field"])
	assert.Equal(t, "Field must be at most 5", got["max_field"])
	assert.Equal(t, "Field must be a valid URL", got["url_field"])
	assert.Equal(t, "Field is invalid", got["other_field"])
}

func TestValidate_UsesJSONFieldNames(t *testing.T) {
	t.Parallel()
	valErrs := collectErrors(t, validationFixture{})

	fields := make([]string, 0, len(valErrs))
	for _, fe := range valErrs {
		fields = append(fields, fe.Field())
	}
	assert.Contains(t, fields, "required_field")
	assert.NotContains(t, fields, "Required")
}

func TestValidateRequest(t *testing.T) {
	t.Parallel()
	logger := zap.NewNop().Sugar()

	validFixture := validationFixture{
		Required: "x",
		Min:      "abcd",
		Max:      "abc",
		URL:      "https://example.com",
		Other:    "red",
	}

	tests := []struct {
		name             string
		req              any
		wantOK           bool
		wantCode         int
		wantBodyContains []string
		wantFields       []string
	}{
		{
			name:     "valid passes",
			req:      validFixture,
			wantOK:   true,
			wantCode: 200,
		},
		{
			name:     "invalid returns 400 with field messages",
			req:      validationFixture{URL: "not-a-url", Other: "blue"},
			wantOK:   false,
			wantCode: 400,
			wantBodyContains: []string{
				"request didn't pass validation",
				"required_field",
				"url_field",
			},
			wantFields: []string{"required_field", "url_field"},
		},
		{
			name:             "non-validation error returns 500",
			req:              "not a struct",
			wantOK:           false,
			wantCode:         500,
			wantBodyContains: []string{"internal server error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			ok := handlers.ValidateRequest(w, logger, tt.req)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantCode, w.Code)

			for _, substr := range tt.wantBodyContains {
				assert.True(t, strings.Contains(w.Body.String(), substr), "body should contain %q", substr)
			}

			if len(tt.wantFields) > 0 {
				var body map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				fields, ok := body["fields"].(map[string]any)
				require.True(t, ok)
				for _, field := range tt.wantFields {
					assert.Contains(t, fields, field)
				}
			}
		})
	}
}
