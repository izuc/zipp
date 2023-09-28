package main

import (
	_ "net/http/pprof"

	"github.com/izuc/zipp/packages/node"
	"github.com/izuc/zipp/plugins"
)

func main() {
	node.Run(
		plugins.Core,
		plugins.Research,
		plugins.UI,
		plugins.WebAPI,
	)
}
