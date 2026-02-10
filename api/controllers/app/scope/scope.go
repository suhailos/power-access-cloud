/*
Copyright 2023.

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

package scope

import (
	"context"

	"github.com/IBM/power-access-cloud/api/apis/app/v1alpha1"
	"github.com/IBM/power-access-cloud/api/controllers/util"
	"github.com/IBM/power-access-cloud/api/internal/pkg/client/kubernetes"
	"github.com/IBM/power-access-cloud/api/internal/pkg/client/platform"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"github.com/IBM/go-sdk-core/v5/core"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/IBM/power-access-cloud/api/internal/pkg/client/powervs"
)

type ControllerScopeParams struct {
	Logger  logr.Logger
	Client  client.Client
	Type    string
	Catalog *v1alpha1.Catalog
	Debug   bool
}

type ControllerScope struct {
	logr.Logger
	Client         client.Client
	Catalog        *v1alpha1.Catalog
	PowerVSClient  *powervs.Client
	PlatformClient *platform.Client
	K8sClient      *kubernetes.Client
}

type CatalogScopeParams struct {
	ControllerScopeParams
}

func NewControllerScope(ctx context.Context, params ControllerScopeParams) (*ControllerScope, error) {
	scope := &ControllerScope{}

	if params.Client == nil {
		return scope, errors.New("client is required when creating a CatalogScope")
	}
	scope.Client = params.Client

	if params.Logger == (logr.Logger{}) {
		params.Logger = zap.New()
	}
	scope.Logger = params.Logger

	if params.Catalog == nil {
		return scope, errors.New("catalog is required when creating a scope for catalog and service controller")
	}
	scope.Catalog = params.Catalog

	platformClient, err := platform.NewClient()
	if err != nil {
		return scope, errors.Wrap(err, "error creating platform services client")
	}
	scope.PlatformClient = platformClient

	var cloudInstanceID, zone, accountID, region string
	switch params.Catalog.Spec.Type {
	case v1alpha1.CatalogTypeVM:
		cloudInstanceID, zone, accountID, err = util.ParsePowerVSCRN(params.Catalog.Spec.VM.CRN)
		if err != nil {
			return scope, err
		}
	case v1alpha1.CatalogTypeAIX:
		cloudInstanceID, zone, accountID, err = util.ParsePowerVSCRN(params.Catalog.Spec.AIX.CRN)
		if err != nil {
			return scope, err
		}
	case v1alpha1.CatalogTypeIBMi:
		cloudInstanceID, zone, accountID, err = util.ParsePowerVSCRN(params.Catalog.Spec.IBMi.CRN)
		if err != nil {
			return scope, err
		}
	case v1alpha1.CatalogTypeK8s:
		cloudInstanceID, zone, accountID, err = util.ParsePowerVSCRN(params.Catalog.Spec.K8s.CRN)
		if err != nil {
			return scope, err
		}
		region = params.Catalog.Spec.K8s.Zone
	}

	// Initialize PowerVS client for VM-based catalogs (VM, AIX, IBMi)
	if params.Catalog.Spec.Type == v1alpha1.CatalogTypeVM ||
		params.Catalog.Spec.Type == v1alpha1.CatalogTypeAIX ||
		params.Catalog.Spec.Type == v1alpha1.CatalogTypeIBMi {
		powerVSClient, err := powervs.NewClient(ctx, powervs.Options{
			AccountID:       accountID,
			CloudInstanceID: cloudInstanceID,
			Zone:            zone,
			Debug:           params.Debug})
		if err != nil {
			return scope, errors.Wrap(err, "failed to create powervs client")
		}
		scope.PowerVSClient = powerVSClient
	}

	// Initialize K8s client for K8s catalogs
	if params.Catalog.Spec.Type == v1alpha1.CatalogTypeK8s {
		k8sClient, err := kubernetes.NewClient(kubernetes.Options{
			AccountID: accountID,
			Region:    region,
		})
		if err != nil {
			return scope, errors.Wrap(err, "failed to create kubernetes client")
		}
		scope.K8sClient = k8sClient
	}

	if params.Debug {
		core.SetLoggingLevel(core.LevelDebug)
	}

	return scope, nil
}
