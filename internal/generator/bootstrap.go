package generator

// generateBootstrap generates the main bootstrap file (glib.gen.go)
func (g *Generator) generateBootstrap() (string, error) {
	// Config is always loaded internally in the DI container
	// Bootstrap never needs config parameter
	data := map[string]any{
		"PackageName":          g.pkgName,
		"ConfigNeedsParameter": false, // Always false - config loaded internally
		"ConfigParameterType":  "",
	}
	return g.executeTemplate("bootstrap.tmpl", data)
}
