package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/revosai/terraform-provider-revos/internal/provider"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/revosai/revos",
	})

	if err != nil {
		log.Fatal(err.Error())
	}
}
