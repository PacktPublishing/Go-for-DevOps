package petstore

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/13/petstore-provider/internal/client"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/13/petstore-provider/internal/client/proto"
)

func resourcePet() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePetCreate,
		ReadContext:   resourcePetRead,
		UpdateContext: resourcePetUpdate,
		DeleteContext: resourcePetDelete,
		Schema:        getPetResourceSchema(),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

// resourcePetCreate creates a pet in the pet store
func resourcePetCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	psClient, err := clientFromMeta(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	pet := &client.Pet{Pet: &pb.Pet{}}
	diags := fillPetFromData(pet, data)
	ids, err := psClient.AddPets(ctx, []*pb.Pet{pet.Pet})
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	data.SetId(ids[0])
	return diags
}

// resourcePetRead finds a pet in the pet store by ID and populate the resource data
func resourcePetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	psClient, err := clientFromMeta(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	pets, err := findPetsInStore(ctx, psClient, findPetsRequest{ID: data.Id()})
	if err != nil {
		return diag.FromErr(err)
	}

	if len(pets) == 0 {
		return nil
	}

	return setDataFromPet(pets[0], data)
}

// resourcePetUpdate updates a pet in the pet store by ID
func resourcePetUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	psClient, err := clientFromMeta(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	pets, err := findPetsInStore(ctx, psClient, findPetsRequest{ID: data.Id()})
	if err != nil {
		return diag.FromErr(err)
	}

	if len(pets) == 0 {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "no pet was found",
				Detail:   "no pet was found when trying to update the pet by ID",
			},
		}
	}

	pet := pets[0]
	diags := fillPetFromData(pet, data)
	if diags.HasError() {
		return diags
	}

	if err := psClient.UpdatePets(ctx, []*pb.Pet{pet.Pet}); err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	return diags
}

// resourcePetDelete deletes a pet in teh pet store by ID
func resourcePetDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	psClient, err := clientFromMeta(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := psClient.DeletePets(ctx, []string{data.Id()}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
