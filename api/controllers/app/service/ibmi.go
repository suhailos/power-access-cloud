package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/IBM/go-sdk-core/v5/core"
	appv1alpha1 "github.com/IBM/power-access-cloud/api/apis/app/v1alpha1"
	"github.com/IBM/power-access-cloud/api/controllers/app/scope"
	"github.com/pkg/errors"
)

var _ Interface = &IBMi{}

type IBMi struct {
	scope *scope.ServiceScope
}

func NewIBMi(scope *scope.ServiceScope) Interface {
	return &IBMi{
		scope: scope,
	}
}

func (s *IBMi) Reconcile(ctx context.Context) error {
	if s.scope.Service.Status.IBMi.InstanceID == "" {
		if err := createIBMiVM(s.scope); err != nil {
			return errors.Wrap(err, "error creating IBMi vm")
		}
	}

	pvmInstance, err := s.scope.PowerVSClient.GetVM(s.scope.Service.Status.IBMi.InstanceID)
	if err != nil {
		return errors.Wrap(err, "error getting IBMi vm")
	}

	updateIBMiStatus(s.scope, pvmInstance)

	return nil
}

func (s *IBMi) Delete(ctx context.Context) (bool, error) {
	if s.scope.Service.Status.IBMi.InstanceID == "" {
		s.scope.Logger.Info("IBMi vm instanceID is empty, nothing to clean up")
		return true, nil
	}

	if err := cleanupIBMiVM(s.scope); err != nil {
		return false, errors.Wrap(err, "error cleaning up IBMi vm")
	}
	s.scope.Service.Status.ClearIBMiStatus()

	return true, nil
}

func cleanupIBMiVM(scope *scope.ServiceScope) error {
	return scope.PowerVSClient.DeleteVM(scope.Service.Status.IBMi.InstanceID)
}

func updateIBMiStatus(scope *scope.ServiceScope, pvmInstance *models.PVMInstance) {
	extractIBMiPVMInstance(scope, pvmInstance)

	switch *pvmInstance.Status {
	case "ACTIVE":
		scope.Service.Status.SetSuccessful()
		scope.Service.Status.State = appv1alpha1.ServiceStateCreated
		scope.Service.Status.AccessInfo = appv1alpha1.IBMiAccessInfoTemplate(
			scope.Service.Status.IBMi.ExternalIPAddress,
			scope.Service.Status.IBMi.IPAddress,
			scope.Service.Status.IBMi.ConsoleURL,
		)
		scope.Service.Status.Message = ""
	case "ERROR":
		scope.Service.Status.State = appv1alpha1.ServiceStateFailed
		if pvmInstance.Fault != nil {
			scope.Service.Status.Message = fmt.Sprintf("IBMi vm creation failed with reason: %s", pvmInstance.Fault.Message)
		}
		scope.Service.Status.AccessInfo = ""
	default:
		scope.Service.Status.State = appv1alpha1.ServiceStateInProgress
		scope.Service.Status.Message = "IBMi vm creation started, will update the access info once vm is ready"
	}
}

func extractIBMiPVMInstance(scope *scope.ServiceScope, pvmInstance *models.PVMInstance) {
	scope.Service.Status.IBMi.InstanceID = *pvmInstance.PvmInstanceID
	for _, nw := range pvmInstance.Networks {
		scope.Service.Status.IBMi.ExternalIPAddress = nw.ExternalIP
		scope.Service.Status.IBMi.IPAddress = nw.IPAddress
	}
	scope.Service.Status.IBMi.State = *pvmInstance.Status
	
	// Extract OS version from operating system info if available
	if pvmInstance.OperatingSystem != nil && *pvmInstance.OperatingSystem != "" {
		scope.Service.Status.IBMi.OSVersion = *pvmInstance.OperatingSystem
	} else {
		// Default to image name if OS info not available
		scope.Service.Status.IBMi.OSVersion = scope.Catalog.Spec.IBMi.Image
	}
	
	// Generate console URL (this would be the PowerVS console URL for the instance)
	// Format: https://cloud.ibm.com/power/instances/{cloudInstanceID}/virtual-servers/{instanceID}
	if scope.Service.Status.IBMi.ConsoleURL == "" {
		cloudInstanceID := scope.PowerVSClient.GetCloudInstanceID()
		scope.Service.Status.IBMi.ConsoleURL = fmt.Sprintf(
			"https://cloud.ibm.com/power/instances/%s/virtual-servers/%s",
			cloudInstanceID,
			scope.Service.Status.IBMi.InstanceID,
		)
	}
}

func createIBMiVM(scope *scope.ServiceScope) error {
	// Check if vm already exists and return if it does
	instances, err := scope.PowerVSClient.GetAllInstance()
	if err != nil {
		return err
	}

	for _, instance := range instances.PvmInstances {
		if *instance.ServerName == scope.Service.ObjectMeta.Name {
			scope.Logger.Info("IBMi vm already exists, hence skipping the vm creation", "name", scope.Service.ObjectMeta.Name)
			scope.Service.Status.IBMi.InstanceID = *instance.PvmInstanceID
			return nil
		}
	}

	ibmiSpec := scope.Catalog.Spec.IBMi
	var networkID string
	
	if ibmiSpec.Network == "" {
		var err error
		networkID, err = getAvailablePubNetwork(scope)
		if err != nil && err != ErroNoPublicNetwork {
			return errors.Wrap(err, "error retrieving available public network in powervs instance")
		} else if err == ErroNoPublicNetwork {
			// Create a public network and use it
			network, err := scope.PowerVSClient.CreateNetwork(&models.NetworkCreate{
				Name:       generateNetworkName(),
				Type:       core.StringPtr("pub-vlan"),
				DNSServers: dnsServers,
			})
			if err != nil {
				return errors.Wrap(err, "error creating public network")
			}
			networkID = *network.NetworkID
		}
	} else {
		nwRef, err := scope.PowerVSClient.GetNetworkByName(ibmiSpec.Network)
		if err != nil {
			return errors.Wrapf(err, "error retrieving network by name %s", ibmiSpec.Network)
		}
		networkID = *nwRef.NetworkID
	}

	imageRef, err := scope.PowerVSClient.GetImageByName(ibmiSpec.Image)
	if err != nil {
		return errors.Wrapf(err, "error retrieving IBMi image by name %s", ibmiSpec.Image)
	}

	memory := float64(ibmiSpec.Capacity.Memory)
	processors, _ := strconv.ParseFloat(ibmiSpec.Capacity.CPU, 64)
	
	createOpts := &models.PVMInstanceCreate{
		ServerName: &scope.Service.Name,
		ImageID:    imageRef.ImageID,
		NetworkIDs: []string{networkID},
		Memory:     &memory,
		Processors: &processors,
		SysType:    ibmiSpec.SystemType,
		ProcType:   &ibmiSpec.ProcessorType,
		UserData:   base64.StdEncoding.EncodeToString([]byte(strings.Join(scope.Service.Spec.SSHKeys, "\n"))),
	}

	// Add license repository if specified
	if ibmiSpec.LicenseRepository != "" {
		createOpts.LicenseRepositoryCapacity = core.Int64Ptr(1)
	}

	pvmInstanceList, err := scope.PowerVSClient.CreateVM(createOpts)
	if err != nil {
		return err
	}
	scope.Service.Status.Message = "IBMi vm creation started, will update the access info once vm is ready"

	if len(*pvmInstanceList) != 1 {
		return errors.New("error creating IBMi vm, expected 1 vm to be created")
	}
	scope.Service.Status.IBMi.InstanceID = *(*pvmInstanceList)[0].PvmInstanceID
	return nil
}

// Made with Bob
