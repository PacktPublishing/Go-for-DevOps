package mem

import (
	"context"
	"sort"
	"strconv"
	"testing"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server/storage"

	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/protobuf/proto"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/proto"
	dpb "google.golang.org/genproto/googleapis/type/date"
)

// This tests we implement the interface.
var _ storage.Data = &Data{}

var pets = []*pb.Pet{
	{
		Id:       "0",
		Name:     "Adam",
		Type:     pb.PetType_PTCanine,
		Birthday: &dpb.Date{Month: 1, Day: 1, Year: 2020},
	},
	{
		Id:       "1",
		Name:     "Becky",
		Type:     pb.PetType_PTFeline,
		Birthday: &dpb.Date{Month: 2, Day: 1, Year: 2020},
	},
	{
		Id:       "2",
		Name:     "Calvin",
		Type:     pb.PetType_PTFeline,
		Birthday: &dpb.Date{Month: 2, Day: 2, Year: 2020},
	},
	{
		Id:       "3",
		Name:     "David",
		Type:     pb.PetType_PTBird,
		Birthday: &dpb.Date{Month: 2, Day: 2, Year: 2021},
	},
	{
		Id:       "4",
		Name:     "Elaine",
		Type:     pb.PetType_PTReptile,
		Birthday: &dpb.Date{Month: 2, Day: 2, Year: 2021},
	},
	{
		Id:       "5",
		Name:     "Elaine",
		Type:     pb.PetType_PTReptile,
		Birthday: &dpb.Date{Month: 2, Day: 3, Year: 2021},
	},
}

// makePets takes the global "pets" var and clones everything in it and puts it into
// a *Data so we have test data.
func makePets() *Data {
	d := New()

	n := []*pb.Pet{}
	for _, p := range pets {
		n = append(n, proto.Clone(p).(*pb.Pet))
	}

	d.AddPets(context.Background(), n)
	return d
}

func TestByNames(t *testing.T) {
	d := makePets()

	got := d.byNames(context.Background(), &pb.SearchPetsReq{Names: []string{"David", "Elaine"}})
	sort.Strings(got)

	want := []string{"3", "4", "5"}
	if diff := pretty.Compare(want, got); diff != "" {
		t.Errorf("TestByNames: -want/+got:\n%s", diff)
	}
}

func TestByTypes(t *testing.T) {
	d := makePets()

	got := d.byTypes(context.Background(), &pb.SearchPetsReq{Types: []pb.PetType{pb.PetType_PTCanine, pb.PetType_PTReptile}})
	sort.Strings(got)

	want := []string{"0", "4", "5"}
	if diff := pretty.Compare(want, got); diff != "" {
		t.Errorf("TestByTypes: -want/+got:\n%s", diff)
	}
}

func TestByBirthdays(t *testing.T) {
	d := makePets()

	got := d.byBirthdays(
		context.Background(),
		&pb.SearchPetsReq{
			BirthdateRange: &pb.DateRange{
				Start: &dpb.Date{Month: 2, Day: 1, Year: 2020},
				End:   &dpb.Date{Month: 2, Day: 3, Year: 2021},
			},
		},
	)
	sort.Strings(got)

	want := []string{"1", "2", "3", "4"}
	if diff := pretty.Compare(want, got); diff != "" {
		t.Errorf("TestByBirthdays: -want/+got:\n%s", diff)
	}
}

func TestDeletePets(t *testing.T) {
	d := makePets()

	deletions := []string{"3", "5", "20"}

	if err := d.DeletePets(context.Background(), deletions); err != nil {
		t.Fatalf("TestDeletePets: got err == %v, want err == nil", err)
	}

	// Don't check the last deletion, it is only there to make sure
	// a non-existent value doesn't do anything.
	for _, id := range deletions[:len(deletions)-1] {
		if _, ok := d.ids[id]; ok {
			t.Errorf("TestDeletePets: found ids[%s]", id)
		}
		i, _ := strconv.Atoi(id)

		if m, ok := d.names[pets[i].Name]; ok {
			if _, ok := m[id]; ok {
				t.Errorf("TestDeletePets: found(%s) in names", id)
			}
		}
		if m, ok := d.types[pets[i].Type]; ok {
			if _, ok := m[id]; ok {
				t.Errorf("TestDeletePets: found(%s) in types", id)
			}
		}

		v := d.birthday.Get(birthdayGet{pets[i]})
		if v != nil {
			if _, ok := v.(birthdays)[id]; ok {
				t.Errorf("TestDeletePets: found(%s) in birthday tree", id)
			}
		}
	}
}

func TestSearchPets(t *testing.T) {
	d := makePets()

	ch := d.SearchPets(
		context.Background(),
		&pb.SearchPetsReq{
			Names: []string{
				"Becky",
				"Calvin",
				"David",
				"Elaine",
			},
			Types: []pb.PetType{
				pb.PetType_PTReptile,
				pb.PetType_PTFeline,
			},
			BirthdateRange: &pb.DateRange{
				Start: &dpb.Date{Month: 2, Day: 2, Year: 2021},
				End:   &dpb.Date{Month: 2, Day: 3, Year: 2021},
			},
		},
	)

	got := []storage.SearchItem{}
	for item := range ch {
		got = append(got, item)
	}

	want := []storage.SearchItem{{Pet: pets[4]}}

	config := pretty.Config{TrackCycles: true}
	if diff := config.Compare(want, got); diff != "" {
		t.Errorf("TestSearchPets: -want/+got:\n%s", diff)
	}
}
