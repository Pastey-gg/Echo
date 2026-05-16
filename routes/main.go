package routes

import "github.com/EvieePy/Echo/state"

type RouteViewI interface {
	LoadRoutes()
}

func LoadRoutes(ctx *state.Context) {
	testView := TestView{ctx}
	testView.LoadRoutes()
}
