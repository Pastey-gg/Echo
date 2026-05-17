//go:generate swag init
package main

import (
	"fmt"

	"github.com/EvieePy/Echo/routes"
	"github.com/EvieePy/Echo/state"
)

// @title Pastey.gg API Documentation
// @license.name AGPL-3.0-or-later
// @license.url https://www.gnu.org/licenses/#AGPL
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
