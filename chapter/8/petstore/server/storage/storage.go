package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/proto"
	dpb "google.golang.org/genproto/googleapis/type/date"
)

// Data represents our data storage.
type Data interface {
	// AddPets adds pet entries into storage.
	AddPets(ctx context.Context, pets []*pb.Pet) error
	// DeletePets deletes pets in storage by their ID. Will not error
	// on IDs not found.
	DeletePets(ctx context.Context, ids []string) error
	// SearchPets searches storage for pet entries that match the
	// filter.
	SearchPets(ctx context.Context, filter *pb.SearchPetsReq) chan SearchItem
}

// SearchItem is an item returned by a search.
type SearchItem struct {
	// Pet is the pet that matched the search filters.
	Pet *pb.Pet
	// Error indicates that there was an error. If set the channel
	// will close after this entry.
	Error error
}

// ValidatePet validates that *pb.Pet has valid fields.
func ValidatePet(p *pb.Pet) error {
	if p.Id != "" {
		return fmt.Errorf("cannot set the Id field")
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return fmt.Errorf("cannot have a pet without a name")
	}

	if p.Type == pb.PetType_PTUnknown {
		return fmt.Errorf("cannot have an unknown pet type")
	}

	_, err := BirthdayToTime(p.Birthday)
	if err != nil {
		return fmt.Errorf("pet(%s) had an error in its birthday: %w", p.Name, err)
	}
	return nil

}

// BirthdayToTime converts the *pb.Pet.Birthday field to a time.Time object.
func BirthdayToTime(d *dpb.Date) (time.Time, error) {
	if d.Month < 1 || d.Month > 12 {
		return time.Time{}, fmt.Errorf("month must be 1-12, was %d", d.Month)
	}
	if d.Day < 1 || d.Day > 31 {
		return time.Time{}, fmt.Errorf("day(%d) was invalid", d.Day)
	}
	t := time.Date(int(d.Year), time.Month(d.Month), int(d.Day), 0, 0, 0, 0, time.UTC)

	if t.Month() != time.Month(d.Month) {
		return time.Time{}, fmt.Errorf("month %v does not have %d days", time.Month(d.Month), d.Day)
	}
	return t, nil
}
