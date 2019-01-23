// Copyright 2019 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rsd

import (
	"github.com/pkg/errors"
)

const (
	// StorageServiceCollectionEntryPoint is a URL path to the StorageServices entry point
	StorageServiceCollectionEntryPoint = "/redfish/v1/StorageServices"
)

// StorageServiceCollection JSON payload structure
type StorageServiceCollection struct {
	OdataContext      string `json:"@odata.context"`
	OdataID           string `json:"@odata.id"`
	OdataType         string `json:"@odata.type"`
	Name              string `json:"Name"`
	MembersOdataCount int    `json:"Members@odata.count"`
	Members           []struct {
		OdataID string `json:"@odata.id"`
	} `json:"Members"`
	Oem struct {
	} `json:"Oem:"`
}

// StorageService JSON payload structure
type StorageService struct {
	OdataContext string `json:"@odata.context"`
	OdataID      string `json:"@odata.id"`
	OdataType    string `json:"@odata.type"`
	ID           int    `json:"Id"`
	Name         string `json:"Name"`
	Status       struct {
		State  string `json:"State"`
		Health string `json:"Health"`
	} `json:"Status"`
	Drives struct {
		OdataID string `json:"@odata.id"`
	} `json:"Drives"`
	Volumes struct {
		OdataID string `json:"@odata.id"`
	} `json:"Volumes"`
	StoragePools struct {
		OdataID string `json:"@odata.id"`
	} `json:"StoragePools"`
	Endpoints struct {
		OdataID string `json:"@odata.id"`
	} `json:"Endpoints"`
	Oem struct {
	} `json:"Oem"`
	Links struct {
		Oem struct {
			IntelRackScale struct {
				ManagedBy []struct {
					OdataID string `json:"@odata.id"`
				} `json:"ManagedBy"`
			} `json:"Intel_RackScale"`
		} `json:"Oem"`
	} `json:"Links"`
}

// GetMembers returns members of StorageService collection
func (collection *StorageServiceCollection) GetMembers(rsd Transport) ([]*StorageService, error) {
	var result []*StorageService
	for _, member := range collection.Members {
		var item StorageService
		err := rsd.Get(member.OdataID, &item)
		if err != nil {
			return nil, errors.Wrapf(err, "Can't query StorageServiceCollection members %s", member.OdataID)
		}

		result = append(result, &item)
	}
	return result, nil
}

// GetVolumeCollection returns VolumeCollection associated with a Storage Service
func (service *StorageService) GetVolumeCollection(rsd Transport) (*VolumeCollection, error) {
	var result VolumeCollection
	err := rsd.Get(service.Volumes.OdataID, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't query StorageService VolumeCollection %s", service.Volumes.OdataID)
	}

	return &result, nil
}
