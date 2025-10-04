// Package exemple is an example package for grouter
package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/azizndao/grouter"
)

func main() {
	router := grouter.NewRouter()

	router.Use(grouter.Logger(),
		grouter.Recovery(func(err any, stack []byte) {
			fmt.Printf("PANIC: %v\n%s\n", err, stack)
		}))

	router.Get("/hello", func(c *grouter.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello World"})
	})

	router.Get("/hello/{name}", func(c *grouter.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": fmt.Sprintf("Hello %s", c.PathValue("name")),
			"query":   c.Query("q"),
		})
	})

	router.Get("/error", func(c *grouter.Ctx) error {
		return grouter.ErrorBadRequest(nil, nil)
	})

	slog.Info("Server started")

	http.ListenAndServe(":8080", router.Handler())
}
