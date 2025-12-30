package cli

// glibConfig represents the .glibrc configuration file structure
type glibConfig struct {
	Version  string `json:"version"`
	Generate struct {
		Output  string `json:"output"`
		Package string `json:"package"`
	} `json:"generate"`
	Make struct {
		Controllers string `json:"controllers"`
		Providers   string `json:"providers"`
		Middleware  string `json:"middleware"`
	} `json:"make"`
	Dev struct {
		Port int `json:"port"`
	} `json:"dev"`
}
