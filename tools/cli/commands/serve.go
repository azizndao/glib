package commands

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

// NewServeCommand creates the "glib serve" command for running development server
func NewServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the development server",
		Long: `Start the glib development server with hot reload support.

Example:
  glib serve
  glib serve --port=3000
  glib serve --host=0.0.0.0`,
		RunE: runServe,
	}

	cmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	cmd.Flags().String("host", "127.0.0.1", "Host to bind to")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	host, _ := cmd.Flags().GetString("host")

	addr := fmt.Sprintf("%s:%d", host, port)

	cmd.Printf("Starting glib development server...\n")
	cmd.Printf("Listening on: http://%s\n", addr)
	cmd.Printf("\nPress Ctrl+C to stop\n\n")

	// TODO: Implement proper application loading
	// 1. Load .env file
	// 2. Bootstrap application
	// 3. Register routes
	// 4. Start server with hot reload

	// For now, just a simple placeholder
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Glib development server is running!\n"))
	})

	if err := http.ListenAndServe(addr, nil); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
