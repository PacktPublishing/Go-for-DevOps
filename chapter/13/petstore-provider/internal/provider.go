package petstore

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/13/petstore-provider/internal/client"
)

// Provider is the entry point for defining the Terraform provider, and will create a new Pet Store provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PETSTORE_HOST", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"petstore_pet": resourcePet(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"petstore_pet": dataSourcePet(),
		},
		ConfigureContextFunc: configure,
	}
}

// configure builds a new Pet Store client the provider will use to interact with the Pet Store service
func configure(_ context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	host, ok := data.Get("host").(string)
	if !ok {
		return nil, diag.Errorf("the host (127.0.0.1:443) must be provided explicitly or via env var PETSTORE_HOST")
	}

	c, err := client.New(host)
	if err != nil {
		return nil, append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create Pet Store client",
			Detail:   "Unable to connect to the Pet Store service",
		})
	}

	return c, diags
}
