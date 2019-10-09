package main

import (
	"log"
	"os"

	"github.com/hashicorp/vault/api"
	artifactory "github.com/jsok/vault-plugin-secrets-artifactory"
)

func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:])

	err := artifactory.Run(apiClientMeta.GetTLSConfig())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
