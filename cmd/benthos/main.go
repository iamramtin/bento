package main

import (
	"github.com/Jeffail/benthos/v3/lib/service"

	// Import all plugins defined within the repo.
	_ "github.com/Jeffail/benthos/v3/public/components/all"
)

func main() {
	service.Run()
}
