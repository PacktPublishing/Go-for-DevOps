package petstore

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/date"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/13/petstore-provider/internal/client"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/13/petstore-provider/internal/client/proto"
)

func dataSourcePet() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePetRead,
		Schema:      getPetDataSchema(),
	}
}

// clientFromMeta casts meta into a Pet Store client or returns an error
func clientFromMeta(meta interface{}) (*client.Client, error) {
	psClient, ok := meta.(*client.Client)
	if !ok {
		return nil, errors.New("meta does not contain a Pet Store client")
	}

	return psClient, nil
}

// dataSourcePetRead finds pets in the pet store given an ID
func dataSourcePetRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	psClient, err := clientFromMeta(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	pets, err := findPetsInStore(ctx, psClient, findPetsRequest{
		Name:     data.Get("name").(string),
		Birthday: data.Get("birthday").(string),
		Type:     PetType(data.Get("type").(string)),
		ID:       data.Get("pet_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// always run
	data.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	if err := data.Set("pets", flattenPets(pets)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func flattenPets(pets []*client.Pet) []interface{} {
	ifacePets := make([]interface{}, len(pets), len(pets))
	for i, pet := range pets {
		ifacePets[i] = petToMap(pet)
	}
	return ifacePets
}

// setDataFromPet populates the resource data from the pet
func setDataFromPet(pet *client.Pet, data *schema.ResourceData) diag.Diagnostics {
	petMap := petToMap(pet)
	var diags diag.Diagnostics

	for k, v := range petMap {
		if err := data.Set(k, v); err != nil {
			diags = append(diags, diag.Errorf("failed to set %s: %s", k, err)...)
		}
	}

	return diags
}

func petToMap(pet *client.Pet) map[string]interface{} {
	return map[string]interface{}{
		"name":     pet.Name,
		"type":     string(protoPetTypeToPetType(pet.Type)),
		"birthday": pet.Birthday().Format("2006-01-02T15:04:05Z07:00"),
		"id":       pet.Id,
	}
}

// fillPetFromData populates a pet from resource data
func fillPetFromData(pet *client.Pet, data *schema.ResourceData) diag.Diagnostics {
	var diags diag.Diagnostics

	pet.Pet.Id = data.Id()
	pet.Pet.Name = data.Get("name").(string)
	pet.Pet.Type = petTypeToProtoPetType(PetType(data.Get("type").(string)))

	if bday, ok := data.Get("birthday").(string); ok {
		t, err := time.Parse(time.RFC3339, bday)
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
		pet.Pet.Birthday = timeToPbDate(t)
	}

	return diags
}

func timeToPbDate(t time.Time) *date.Date {
	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}
}

type findPetsRequest struct {
	ID       string
	Name     string
	Type     PetType
	Birthday string
}

// findPetInStore searches the pet store for a pet that matches the custom resource pet.
func findPetsInStore(ctx context.Context, psClient *client.Client, req findPetsRequest) ([]*client.Pet, error) {
	searchReq := &pb.SearchPetsReq{}
	if string(req.Type) != "" {
		searchReq.Types = []pb.PetType{petTypeToProtoPetType(req.Type)}
	}

	if req.Name != "" {
		searchReq.Names = []string{req.Name}
	}

	petsChan, err := psClient.SearchPets(ctx, searchReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed searching for pet")
	}

	var pets []*client.Pet
	for pet := range petsChan {
		pet := pet
		if pet.Error() != nil {
			return nil, pet.Error()
		}

		if req.ID == "" || pet.Id == req.ID {
			pets = append(pets, &pet)
		}
	}

	return pets, nil
}

func petTypeToProtoPetType(petType PetType) pb.PetType {
	switch petType {
	case DogPetType:
		return pb.PetType_PTCanine
	case CatPetType:
		return pb.PetType_PTFeline
	case BirdPetType:
		return pb.PetType_PTBird
	default:
		return pb.PetType_PTReptile
	}
}

func protoPetTypeToPetType(pbPetType pb.PetType) PetType {
	switch pbPetType {
	case pb.PetType_PTCanine:
		return DogPetType
	case pb.PetType_PTFeline:
		return CatPetType
	case pb.PetType_PTBird:
		return BirdPetType
	default:
		return ReptilePetType
	}
}
