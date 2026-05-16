package main

import (
	"github.com/EvieePy/Echo/routes"
	"github.com/EvieePy/Echo/state"
)

func main() {
	ctx := state.NewContext()

	// Load Routes...
	routes.LoadRoutes(ctx)

	// Default start server stuff...
	if err := ctx.Server.Start(":1323"); err != nil {
		ctx.Server.Logger.Error("failed to start server", "error", err)
	}
}
