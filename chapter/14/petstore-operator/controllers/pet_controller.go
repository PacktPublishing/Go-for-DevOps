/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/date"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	petstorev1 "github.com/PacktPublishing/Go-for-DevOps/chapter/14/petstore-operator/api/v1alpha1"
	psclient "github.com/PacktPublishing/Go-for-DevOps/chapter/14/petstore-operator/client"
	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/14/petstore-operator/client/proto"
)

const (
	PetFinalizer = "pet.petstore.example.com"
)

// PetReconciler reconciles a Pet object
type PetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=petstore.example.com,resources=pets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=petstore.example.com,resources=pets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=petstore.example.com,resources=pets/finalizers,verbs=update

// Reconcile moves the current state of the pet to be the desired state described in the pet.spec.
func (r *PetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, errResult error) {
	logger := log.FromContext(ctx)

	pet := &petstorev1.Pet{}
	if err := r.Get(ctx, req.NamespacedName, pet); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("object was not found")
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to fetch pet from API server")
		// this will cause this pet resource to be requeued
		return ctrl.Result{}, err
	}

	helper, err := patch.NewHelper(pet, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create patch helper")
	}
	defer func() {
		// patch the resource stored in the API server if something changed.
		if err := helper.Patch(ctx, pet); err != nil {
			errResult = err
		}
	}()

	if pet.DeletionTimestamp.IsZero() {
		// the pet is not marked for delete reconcile desired state
		return r.ReconcileNormal(ctx, pet)
	}

	// pet has been marked for delete, so delete from the petstore
	return r.ReconcileDelete(ctx, pet)
}

// ReconcileNormal will ensure the finalizer and save the desired state to the petstore.
func (r *PetReconciler) ReconcileNormal(ctx context.Context, pet *petstorev1.Pet) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx).WithValues("pet", pet.Spec.Name, "id", pet.Status.ID)

	// add a finalizer to ensure we clean up before K8s garbage collects
	logger.Info("ensuring finalizer")
	controllerutil.AddFinalizer(pet, PetFinalizer)

	psc, err := getPetstoreClient()
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "unable to construct petstore client")
	}

	logger.Info("finding pets in store")
	psPet, err := findPetInStore(ctx, psc, pet)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed trying to find pet in pet store")
	}

	if psPet == nil {
		logger.Info("psPet was not nil")
		// no pet was found, so we must create a pet in the pet store
		err := createPetInStore(ctx, pet, psc)
		return ctrl.Result{}, err
	}

	// pet was found, so we need to update the pet in the pet store
	if err := updatePetInStore(ctx, psc, pet, psPet.Pet); err != nil {
		logger.Info("updating pet in store")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// ReconcileDelete deletes the pet from the petstore and removes the finalizer.
func (r *PetReconciler) ReconcileDelete(ctx context.Context, pet *petstorev1.Pet) (ctrl.Result, error) {
	psc, err := getPetstoreClient()
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "unable to construct petstore client")
	}

	if pet.Status.ID != "" {
		if err := psc.DeletePets(ctx, []string{pet.Status.ID}); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to delete pet")
		}
	}

	// remove finalizer, so K8s can garbage collect the resource.
	controllerutil.RemoveFinalizer(pet, PetFinalizer)
	return ctrl.Result{}, nil
}

func createPetInStore(ctx context.Context, pet *petstorev1.Pet, psc *psclient.Client) error {
	pbPet := &pb.Pet{
		Name:     pet.Spec.Name,
		Type:     petTypeToProtoPetType(pet.Spec.Type),
		Birthday: timeToPbDate(pet.Spec.Birthday),
	}
	ids, err := psc.AddPets(ctx, []*pb.Pet{pbPet})

	if err != nil {
		return errors.Wrap(err, "failed to create new pet in store")
	}
	pet.Status.ID = ids[0]
	return nil
}

func updatePetInStore(ctx context.Context, psc *psclient.Client, pet *petstorev1.Pet, pbPet *pb.Pet) error {
	pbPet.Name = pet.Spec.Name
	pbPet.Type = petTypeToProtoPetType(pet.Spec.Type)
	pbPet.Birthday = timeToPbDate(pet.Spec.Birthday)
	if err := psc.UpdatePets(ctx, []*pb.Pet{pbPet}); err != nil {
		return errors.Wrap(err, "failed to update the pet in the store")
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&petstorev1.Pet{}).
		Complete(r)
}

// findPetInStore searches the pet store for a pet that matches the custom resource pet.
func findPetInStore(ctx context.Context, psc *psclient.Client, pet *petstorev1.Pet) (*psclient.Pet, error) {
	petsChan, err := psc.SearchPets(ctx, &pb.SearchPetsReq{
		Names: []string{pet.Spec.Name},
		Types: []pb.PetType{petTypeToProtoPetType(pet.Spec.Type)},
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed searching for pet")
	}

	for pbPet := range petsChan {
		if pbPet.Error() != nil {
			logger := ctrl.LoggerFrom(ctx)
			logger.Error(err, "search chan error")
			continue
		}

		if pbPet.Id == pet.Status.ID {
			return &pbPet, nil
		}
	}

	return nil, nil
}

func petTypeToProtoPetType(petType petstorev1.PetType) pb.PetType {
	switch petType {
	case petstorev1.DogPetType:
		return pb.PetType_PTCanine
	case petstorev1.CatPetType:
		return pb.PetType_PTFeline
	case petstorev1.BirdPetType:
		return pb.PetType_PTBird
	default:
		return pb.PetType_PTReptile
	}
}

func timeToPbDate(t metav1.Time) *date.Date {
	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}
}

func getPetstoreClient() (*psclient.Client, error) {
	return psclient.New("petstore-service.petstore.svc.cluster.local:6742")
}
