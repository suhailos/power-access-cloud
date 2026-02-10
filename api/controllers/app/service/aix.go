package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/IBM-Cloud/power-go-client/power/models"
	appv1alpha1 "github.com/IBM/power-access-cloud/api/apis/app/v1alpha1"
	"github.com/IBM/power-access-cloud/api/controllers/app/scope"
	"github.com/pkg/errors"
)

var _ Interface = &AIX{}

type AIX struct {
	scope *scope.ServiceScope
}

func NewAIX(scope *scope.ServiceScope) Interface {
	return &AIX{
		scope: scope,
	}
}

func (s *AIX) Reconcile(ctx context.Context) error {
	if s.scope.Service.Status.AIX.InstanceID == "" {
		if err := createAIXVM(s.scope); err != nil {
			return errors.Wrap(err, "error creating AIX vm")
		}
	}

	pvmInstance, err := s.scope.PowerVSClient.GetVM(s.scope.Service.Status.AIX.InstanceID)
	if err != nil {
		return errors.Wrap(err, "error getting AIX vm")
	}

	updateAIXStatus(s.scope, pvmInstance)

	return nil
}

func (s *AIX) Delete(ctx context.Context) (bool, error) {
	if s.scope.Service.Status.AIX.InstanceID == "" {
		s.scope.Logger.Info("AIX vm instanceID is empty, nothing to clean up")
		return true, nil
	}

	if err := cleanupAIXVM(s.scope); err != nil {
		return false, errors.Wrap(err, "error cleaning up AIX vm")
	}
	s.scope.Service.Status.ClearAIXStatus()

	return true, nil
}

func cleanupAIXVM(scope *scope.ServiceScope) error {
	return scope.PowerVSClient.DeleteVM(scope.Service.Status.AIX.InstanceID)
}

func updateAIXStatus(scope *scope.ServiceScope, pvmInstance *models.PVMInstance) {
	extractAIXPVMInstance(scope, pvmInstance)

	switch *pvmInstance.Status {
	case "ACTIVE":
		scope.Service.Status.SetSuccessful()
		scope.Service.Status.State = appv1alpha1.ServiceStateCreated
		scope.Service.Status.AccessInfo = appv1alpha1.AIXAccessInfoTemplate(
			scope.Service.Status.AIX.ExternalIPAddress,
			scope.Service.Status.AIX.IPAddress,
			scope.Service.Status.AIX.OSVersion,
		)
		scope.Service.Status.Message = ""
	case "ERROR":
		scope.Service.Status.State = appv1alpha1.ServiceStateFailed
		if pvmInstance.Fault != nil {
			scope.Service.Status.Message = fmt.Sprintf("AIX vm creation failed with reason: %s", pvmInstance.Fault.Message)
		}
		scope.Service.Status.AccessInfo = ""
	default:
		scope.Service.Status.State = appv1alpha1.ServiceStateInProgress
		scope.Service.Status.Message = "AIX vm creation started, will update the access info once vm is ready"
	}
}

func extractAIXPVMInstance(scope *scope.ServiceScope, pvmInstance *models.PVMInstance) {
	scope.Service.Status.AIX.InstanceID = *pvmInstance.PvmInstanceID
	for _, nw := range pvmInstance.Networks {
		scope.Service.Status.AIX.ExternalIPAddress = nw.ExternalIP
		scope.Service.Status.AIX.IPAddress = nw.IPAddress
	}
	scope.Service.Status.AIX.State = *pvmInstance.Status
	
	// Extract OS version from operating system info if available
	if pvmInstance.OperatingSystem != nil && *pvmInstance.OperatingSystem != "" {
		scope.Service.Status.AIX.OSVersion = *pvmInstance.OperatingSystem
	} else {
		// Default to image name if OS info not available
		scope.Service.Status.AIX.OSVersion = scope.Catalog.Spec.AIX.Image
	}
}

func createAIXVM(scope *scope.ServiceScope) error {
	// Check if vm already exists and return if it does
	instances, err := scope.PowerVSClient.GetAllInstance()
	if err != nil {
		return err
	}

	for _, instance := range instances.PvmInstances {
		if *instance.ServerName == scope.Service.ObjectMeta.Name {
			scope.Logger.Info("AIX vm already exists, hence skipping the vm creation", "name", scope.Service.ObjectMeta.Name)
			scope.Service.Status.AIX.InstanceID = *instance.PvmInstanceID
			return nil
		}
	}

	aixSpec := scope.Catalog.Spec.AIX
	var networkID string
	
	if aixSpec.Network == "" {
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
		nwRef, err := scope.PowerVSClient.GetNetworkByName(aixSpec.Network)
		if err != nil {
			return errors.Wrapf(err, "error retrieving network by name %s", aixSpec.Network)
		}
		networkID = *nwRef.NetworkID
	}

	imageRef, err := scope.PowerVSClient.GetImageByName(aixSpec.Image)
	if err != nil {
		return errors.Wrapf(err, "error retrieving AIX image by name %s", aixSpec.Image)
	}

	memory := float64(aixSpec.Capacity.Memory)
	processors, _ := strconv.ParseFloat(aixSpec.Capacity.CPU, 64)
	
	createOpts := &models.PVMInstanceCreate{
		ServerName: &scope.Service.Name,
		ImageID:    imageRef.ImageID,
		NetworkIDs: []string{networkID},
		Memory:     &memory,
		Processors: &processors,
		SysType:    aixSpec.SystemType,
		ProcType:   &aixSpec.ProcessorType,
		UserData:   base64.StdEncoding.EncodeToString([]byte(strings.Join(scope.Service.Spec.SSHKeys, "\n"))),
	}

	pvmInstanceList, err := scope.PowerVSClient.CreateVM(createOpts)
	if err != nil {
		return err
	}
	scope.Service.Status.Message = "AIX vm creation started, will update the access info once vm is ready"

	if len(*pvmInstanceList) != 1 {
		return errors.New("error creating AIX vm, expected 1 vm to be created")
	}
	scope.Service.Status.AIX.InstanceID = *(*pvmInstanceList)[0].PvmInstanceID
	return nil
}

// Made with Bob
