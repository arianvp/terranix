package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/zclconf/go-cty/cty"

	"github.com/zclconf/go-cty/cty/json"

	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/providers"
)

func providerFactory(meta discovery.PluginMeta) providers.Factory {
	return func() (providers.Interface, error) {
		client := tfplugin.Client(meta)
		// Request the RPC client so we can get the provider
		// so we can build the actual RPC-implemented provider.
		rpcClient, err := client.Client()
		if err != nil {
			return nil, err
		}

		raw, err := rpcClient.Dispense(tfplugin.ProviderPluginName)
		if err != nil {
			return nil, err
		}

		// store the client so that the plugin can kill the child process
		p := raw.(*tfplugin.GRPCProvider)
		p.PluginClient = client
		return p, nil
	}
}

func readValue(filePath string, schema providers.Schema) (cty.Value, error) {

	jsonFile, err := os.Open(filePath)
	if err != nil {
		return cty.NilVal, err
	}

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return cty.NilVal, err
	}

	value, err := json.Unmarshal(byteValue, schema.Block.ImpliedType())
	if err != nil {
		return cty.NilVal, err
	}

	value, err = schema.Block.CoerceValue(value)
	if err != nil {
		return cty.NilVal, err
	}
	return value, nil
}

func main() {
	path := os.Args[1]
	meta := discovery.PluginMeta{
		Name:    "terraform-provider-digitalocean", //"terraform-provider-aws",
		Version: "2.23.0",                          //"2.23.0",
		Path:    path,                              //"./result-bin/bin/terraform-provider-aws_v2.23.0",
	}
	provider, err := providerFactory(meta)()
	defer provider.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	schemaResponse := provider.GetSchema()

	spew.Dump(schemaResponse)

}
