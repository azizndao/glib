package generator

// generateErrors generates error handling helpers (errors.gen.go)
func (g *Generator) generateErrors() (string, error) {
	data := map[string]any{
		"PackageName": g.pkgName,
	}
	return g.executeTemplate("errors.tmpl", data)
}
