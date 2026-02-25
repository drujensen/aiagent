package entities

import (
	"fmt"
	"regexp"
)

type Skill struct {
	Name          string            `json:"name"`
	Summary       string            `json:"description"`
	Path          string            `json:"path"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  []string          `json:"allowed_tools,omitempty"`
	// Content is lazy-loaded
	Content string `json:"-"`
}

// Validate checks if the skill meets the Agent Skills specification requirements
func (s *Skill) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if len(s.Name) > 64 {
		return fmt.Errorf("skill name must be 64 characters or less")
	}
	if s.Summary == "" {
		return fmt.Errorf("skill description is required")
	}
	if len(s.Summary) > 1024 {
		return fmt.Errorf("skill description must be 1024 characters or less")
	}

	// Name validation: lowercase letters, numbers, hyphens only
	// Must not start or end with hyphen, no consecutive hyphens
	validName := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	if !validName.MatchString(s.Name) {
		return fmt.Errorf("skill name must contain only lowercase letters, numbers, and hyphens, and cannot start or end with a hyphen or have consecutive hyphens")
	}

	if s.Compatibility != "" && len(s.Compatibility) > 500 {
		return fmt.Errorf("compatibility field must be 500 characters or less")
	}

	return nil
}

// Implement the list.Item interface for BubbleTea list
func (s *Skill) FilterValue() string {
	return fmt.Sprintf("%s %s", s.Name, s.Summary)
}

func (s *Skill) Title() string {
	return s.Name
}

func (s *Skill) Description() string {
	// Truncate summary if too long for display
	desc := s.Summary
	if len(desc) > 100 {
		desc = desc[:97] + "..."
	}
	if s.License != "" {
		desc += fmt.Sprintf(" (License: %s)", s.License)
	}
	return desc
}
