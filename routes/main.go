package routes

import "github.com/EvieePy/Echo/state"

type RouteViewI interface {
	LoadRoutes()
}

func LoadRoutes(ctx *state.Context) {
	views := []RouteViewI{
		&MetaView{ctx},
		&PasteView{ctx},
		&SecurityView{ctx},
		&DocsView{ctx},
	}

	for _, v := range views {
		v.LoadRoutes()
	}
}
