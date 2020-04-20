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

func blockToNixOSModule() {

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
	providerPath := os.Args[2]
	oldState := os.Args[3]
	newState := os.Args[4]
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
	providerSchema := schemaResponse.Provider

	value, err := readValue(providerPath, providerSchema)
	configureRequest := providers.ConfigureRequest{
		TerraformVersion: "mock",
		Config:           value,
	}
	configureResponse := provider.Configure(configureRequest)

	if err = configureResponse.Diagnostics.Err(); err != nil {
		fmt.Println(err)
		return
	}

	dropletSchema := schemaResponse.ResourceTypes["digitalocean_droplet"]

	priorState, err := readValue(oldState, dropletSchema)
	if err != nil {
		fmt.Println(err)
		return
	}

	proposedState, err := readValue(newState, dropletSchema)
	if err != nil {
		fmt.Println(err)
		return
	}

	// so apparently I need to provide the priorState and proposedState. The
	// format of state seems undocumented on first sight though. as it's not the
	// same as the config

	nullConfig := cty.NullVal(dropletSchema.Block.ImpliedType()) // providers shouldn't use the config. if they do they're buggy :P
	planResponse := provider.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName:         "digitalocean_droplet",
		PriorState:       priorState,
		ProposedNewState: proposedState,
		Config:           nullConfig,
		PriorPrivate:     nil,
	})

	if err = planResponse.Diagnostics.Err(); err != nil {
		fmt.Println(err)
		return
	}

	spew.Dump(planResponse)

	/*
		applyResponse := provider.ApplyResourceChange(providers.ApplyResourceChangeRequest{
			TypeName:       "digitalocean_droplet",
			PriorState:     priorState,
			PlannedState:   planResponse.PlannedState,
			Config:         nullConfig,
			PlannedPrivate: planResponse.PlannedPrivate,
		})

		if err = applyResponse.Diagnostics.Err(); err != nil {
			fmt.Println(err)
			return
		}*/
}
