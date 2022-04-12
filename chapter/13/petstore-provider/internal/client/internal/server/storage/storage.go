package storage

import (
	"context"
	"strings"
	"time"

	dpb "google.golang.org/genproto/googleapis/type/date"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/13/petstore-provider/internal/client/internal/server/errors"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/13/petstore-provider/internal/client/proto"
)

// Data represents our data storage.
type Data interface {
	// AddPets adds pet entries into storage.
	AddPets(ctx context.Context, pets []*pb.Pet) error
	// UpdatePets updates pet entries in storage.
	UpdatePets(ctx context.Context, pets []*pb.Pet) error
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
func ValidatePet(ctx context.Context, p *pb.Pet, forUpdate bool) error {
	if forUpdate && p.Id == "" {
		return errors.New(ctx, "updates must have the Id field set")
	} else {
		if !forUpdate && p.Id != "" {
			return errors.New(ctx, "cannot set the Id field")
		}
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New(ctx, "cannot have a pet without a name")
	}

	if p.Type == pb.PetType_PTUnknown {
		return errors.New(ctx, "cannot have an unknown pet type")
	}

	_, err := BirthdayToTime(ctx, p.Birthday)
	if err != nil {
		return errors.Errorf(ctx, "pet(%s) had an error in its birthday: %w", p.Name, err)
	}
	return nil

}

// BirthdayToTime converts the *pb.Pet.Birthday field to a time.Time object.
func BirthdayToTime(ctx context.Context, d *dpb.Date) (time.Time, error) {
	if d.Month < 1 || d.Month > 12 {
		return time.Time{}, errors.Errorf(ctx, "month must be 1-12, was %d", d.Month)
	}
	if d.Day < 1 || d.Day > 31 {
		return time.Time{}, errors.Errorf(ctx, "day(%d) was invalid", d.Day)
	}
	t := time.Date(int(d.Year), time.Month(d.Month), int(d.Day), 0, 0, 0, 0, time.UTC)

	if t.Month() != time.Month(d.Month) {
		return time.Time{}, errors.Errorf(ctx, "month %v does not have %d days", time.Month(d.Month), d.Day)
	}
	return t, nil
}
