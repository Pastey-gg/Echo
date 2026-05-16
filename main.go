package main

import (
	"fmt"

	"github.com/EvieePy/Echo/routes"
	"github.com/EvieePy/Echo/state"
)

func main() {
	ctx := state.NewContext()

	// Load Routes...
	routes.LoadRoutes(ctx)

	// Start Server...
	conf := ctx.Config
	host := conf.General.Host
	port := conf.General.Port

	if err := ctx.Server.Start(fmt.Sprintf("%v:%v", host, port)); err != nil {
		ctx.Logger.Errorf("Unable to start Echo Backend: %v", err)
	}
}
