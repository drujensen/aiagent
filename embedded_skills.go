package main

import "embed"

// embeddedSkillsFS embeds the tracked skills/ directory at the repo root.
// go:embed patterns cannot escape the directory of the source file that
// declares them, so this accessor must live at the repo root (alongside
// main.go) rather than in a nested package - internal/impl/repositories
// receives the resulting embed.FS by constructor injection instead.
//
//go:embed skills
var embeddedSkillsFS embed.FS
