package generator

// generateBootstrap generates the main bootstrap file (glib.gen.go)
func (g *Generator) generateBootstrap() (string, error) {
	data := map[string]any{
		"PackageName": g.pkgName,
	}
	return g.executeTemplate("bootstrap.tmpl", data)
}
