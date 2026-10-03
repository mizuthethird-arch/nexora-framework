package main

import (
	"context"
	"log"

	"github.com/mizuthethird-arch/nexora-framework/internal/nexora"
)

func main() {
	ctx := context.Background()

	app := nexora.New()

	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
