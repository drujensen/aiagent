package uicontrollers

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// stubSkillService is a minimal SkillService stand-in - this controller
// only calls ListSkills, so a full mock isn't needed.
type stubSkillService struct {
	skills []*entities.Skill
	err    error
}

func (s *stubSkillService) ListSkills(ctx context.Context) ([]*entities.Skill, error) {
	return s.skills, s.err
}

func (s *stubSkillService) GetSkillContent(ctx context.Context, skillName string) (string, error) {
	return "", nil
}

// parseUITemplates parses the real templates/*.html files (via relative
// filesystem path, since the ui package's go:embed var is unexported and
// this test lives in a sibling package). The funcMap stand-ins below exist
// only so unrelated templates in the same glob - which reference these
// functions - parse successfully; none of them are invoked when this test
// only executes "layout" with ContentTemplate "skills_list_content".
func parseUITemplates(t *testing.T) *template.Template {
	t.Helper()
	funcMap := template.FuncMap{
		"renderMarkdown":   func(markdown string) (template.HTML, error) { return template.HTML(markdown), nil },
		"formatToolResult": func(toolName, result, diff, arguments string) template.HTML { return template.HTML(result) },
		"formatToolName":   func(toolName, arguments string) string { return toolName },
		"inArray": func(value string, array []string) bool {
			return slices.Contains(array, value)
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"formatNumber": func(num int) string {
			return ""
		},
		"collectModelNames": func(models []entities.ModelPricing) []string {
			return nil
		},
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(filepath.Join("..", "templates", "*.html"))
	assert.NoError(t, err)
	return tmpl
}

// TestSkillController_ListSkillsHandler_RendersDiscoveredSkills is Phase
// 7's named functional assertion for the target feature: the web UI
// skills page renders every discovered skill's name and description.
func TestSkillController_ListSkillsHandler_RendersDiscoveredSkills(t *testing.T) {
	tmpl := parseUITemplates(t)
	stub := &stubSkillService{skills: []*entities.Skill{
		{Name: "research", Summary: "Investigate a problem before design work begins"},
		{Name: "design", Summary: "Turn research into a technical design"},
	}}
	controller := NewSkillController(zap.NewNop(), tmpl, stub)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/skills", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := controller.ListSkillsHandler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "research")
	assert.Contains(t, body, "Investigate a problem before design work begins")
	assert.Contains(t, body, "design")
	assert.Contains(t, body, "Turn research into a technical design")
}

func TestSkillController_ListSkillsHandler_EmptyState(t *testing.T) {
	tmpl := parseUITemplates(t)
	stub := &stubSkillService{skills: nil}
	controller := NewSkillController(zap.NewNop(), tmpl, stub)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/skills", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := controller.ListSkillsHandler(c)
	assert.NoError(t, err)
	assert.Contains(t, rec.Body.String(), "No skills found")
}

func TestSkillController_ListSkillsHandler_ServiceError(t *testing.T) {
	tmpl := parseUITemplates(t)
	stub := &stubSkillService{err: assert.AnError}
	controller := NewSkillController(zap.NewNop(), tmpl, stub)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/skills", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := controller.ListSkillsHandler(c)
	assert.NoError(t, err) // handler itself returns nil; the error is reflected in the response
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
