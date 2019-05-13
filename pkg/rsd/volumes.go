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
	"net/url"

	"github.com/pkg/errors"
)

// VolumeCollection JSON payload structure
type VolumeCollection struct {
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

// Volume JSON payload structure
type Volume struct {
	OdataContext       string `json:"@odata.context"`
	OdataID            string `json:"@odata.id"`
	OdataType          string `json:"@odata.type"`
	Name               string `json:"Name"`
	Description        string `json:"Description"`
	ID                 string `json:"Id"`
	Manufacturer       string `json:"Manufacturer"`
	Model              string `json:"Model"`
	VolumeType         string `json:"VolumeType"`
	Encrypted          bool   `json:"Encrypted"`
	EncryptionTypes    string `json:"EncryptionTypes"`
	BlockSizeBytes     int    `json:"BlockSizeBytes"`
	CapacityBytes      int64  `json:"CapacityBytes"`
	OptimumIOSizeBytes int    `json:"OptimumIOSizeBytes"`
	Status             struct {
		State  string `json:"State"`
		Health string `json:"Health"`
	} `json:"Status"`
	Identifiers []struct {
		DurableNameFormat string `json:"DurableNameFormat"`
		DurableName       string `json:"DurableName"`
	} `json:"Identifiers"`
	Capacity struct {
		Data struct {
			AllocatedBytes int64 `json:"AllocatedBytes"`
		} `json:"Data"`
	} `json:"Capacity"`
	CapacitySources []struct {
		ProvidedCapacity struct {
			Data struct {
				AllocatedBytes int64 `json:"AllocatedBytes"`
			} `json:"Data"`
		} `json:"ProvidedCapacity"`
		ProvidingDrives []map[string]string `json:"ProvidingDrives"`
		ProvidingPools  []map[string]string `json:"ProvidingPools"`
	} `json:"CapacitySources"`
	AccessCapabilities []string `json:"AccessCapabilities"`
	ReplicaInfos       []struct {
		ReplicaType string `json:"ReplicaType"`
		Replica     struct {
			OdataID string `json:"@odata.id"`
		} `json:"Replica"`
	} `json:"ReplicaInfos"`
	Links struct {
		Drives []map[string]string `json:"Drives"`
		Oem    struct {
			IntelRackScale struct {
				OdataType string            `json:"@odata.type"`
				Endpoints []endPointOdataID `json:"Endpoints"`
			} `json:"Intel_RackScale"`
		} `json:"Oem"`
	} `json:"Links"`
	Actions struct {
	} `json:"Actions"`
	Operations []struct {
		OperationName      string `json:"OperationName"`
		PercentageComplete int    `json:"PercentageComplete"`
	} `json:"Operations"`
	Oem struct {
		IntelRackScale struct {
			OdataType     string `json:"@odata.type"`
			Bootable      bool   `json:"Bootable"`
			EraseOnDetach bool   `json:"EraseOnDetach"`
			Erased        bool   `json:"Erased"`
			Image         string `json:"Image"`
			Metrics       struct {
				OdataID string `json:"@odata.id"`
			} `json:"Metrics"`
		} `json:"Intel_RackScale"`
	} `json:"Oem"`
}

// NewVolume creates new volume
func (collection *VolumeCollection) NewVolume(rsd Transport, capacity int64) (*Volume, error) {
	data := map[string]int64{"CapacityBytes": capacity}
	header, err := rsd.Post(collection.OdataID, data, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Can't create new Volume")
	}

	location := header.Get("Location")
	if location == "" {
		return nil, errors.Errorf("No 'Location' header found: %s", collection.OdataID)
	}

	locURL, err := url.Parse(location)
	if err != nil {
		return nil, errors.Errorf("Can't parse location url %s for new volume", location)
	}

	var volume Volume
	err = rsd.Get(locURL.EscapedPath(), &volume)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't query new volume url: %s", locURL.EscapedPath())
	}

	return &volume, nil
}

// GetMembers returns members of Volume collection
func (collection *VolumeCollection) GetMembers(rsd Transport) ([]*Volume, error) {
	var result []*Volume
	for _, member := range collection.Members {
		var item Volume
		err := rsd.Get(member.OdataID, &item)
		if err != nil {
			return nil, errors.Wrapf(err, "Can't query VolumeCollection members %s", member.OdataID)
		}

		result = append(result, &item)
	}
	return result, nil
}

// Delete deletes volume
func (volume *Volume) Delete(rsd Transport) error {
	_, err := rsd.Delete(volume.OdataID, map[string]string{}, nil)
	if err != nil {
		return errors.Wrapf(err, "Can't delete Volume %s", volume.Name)
	}
	return nil
}

// GetEndPoints returns List of EndPoints associated with a Volume
func (volume *Volume) GetEndPoints(rsd Transport) ([]*EndPoint, error) {
	return GetEndPoints(rsd, volume.Links.Oem.IntelRackScale.Endpoints)
}
