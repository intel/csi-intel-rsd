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

import "github.com/pkg/errors"

// StoragePoolCollection JSON payload structure
type StoragePoolCollection struct {
	OdataContext string `json:"@odata.context"`
	OdataID      string `json:"@odata.id"`
	OdataType    string `json:"@odata.type"`
	Description  string `json:"Description"`
	Members      []struct {
		OdataID string `json:"@odata.id"`
	} `json:"Members"`
	MembersOdataCount int    `json:"Members@odata.count"`
	Name              string `json:"Name"`
	Oem               struct {
		IntelRackScale struct {
			TaggedValues struct {
			} `json:"TaggedValues"`
			OdataType string `json:"@odata.type"`
		} `json:"Intel_RackScale"`
	} `json:"Oem"`
}

// StoragePool JSON payload structure
type StoragePool struct {
	OdataContext   string `json:"@odata.context"`
	OdataID        string `json:"@odata.id"`
	OdataType      string `json:"@odata.type"`
	AllocatedPools struct {
		OdataID string `json:"@odata.id"`
	} `json:"AllocatedPools"`
	AllocatedVolumes struct {
		OdataID string `json:"@odata.id"`
	} `json:"AllocatedVolumes"`
	BlockSizeBytes int `json:"BlockSizeBytes"`
	Capacity       struct {
		Data struct {
			AllocatedBytes  int64 `json:"AllocatedBytes"`
			ConsumedBytes   int   `json:"ConsumedBytes"`
			GuaranteedBytes int64 `json:"GuaranteedBytes"`
		} `json:"Data"`
	} `json:"Capacity"`
	CapacitySources []struct {
		OdataID string `json:"@odata.id"`
	} `json:"CapacitySources"`
	Description string `json:"Description"`
	ID          string `json:"Id"`
	Identifier  struct {
		DurableName       string `json:"DurableName"`
		DurableNameFormat string `json:"DurableNameFormat"`
	} `json:"Identifier"`
	Name string `json:"Name"`
	Oem  struct {
		IntelRackScale struct {
			TaggedValues struct {
			} `json:"TaggedValues"`
			OdataType string `json:"@odata.type"`
		} `json:"Intel_RackScale"`
	} `json:"Oem"`
	Status struct {
		Health       string `json:"Health"`
		HealthRollup string `json:"HealthRollup"`
		State        string `json:"State"`
	} `json:"Status"`
}

// GetMembers returns members of StoragePool collection
func (collection *StoragePoolCollection) GetMembers(rsd Transport) ([]*StoragePool, error) {
	var result []*StoragePool
	for _, member := range collection.Members {
		var item StoragePool
		err := rsd.Get(member.OdataID, &item)
		if err != nil {
			return nil, errors.Wrapf(err, "Can't query StoragePoolCollection members %s", member.OdataID)
		}

		result = append(result, &item)
	}
	return result, nil
}
