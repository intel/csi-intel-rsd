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

import "strings"

// EndPoint JSON payload structure
type EndPoint struct {
	OdataContext string `json:"@odata.context"`
	OdataID      string `json:"@odata.id"`
	OdataType    string `json:"@odata.type"`
	Actions      struct {
		Oem struct {
		} `json:"Oem"`
	} `json:"Actions"`
	ConnectedEntities []struct {
		EntityLink struct {
			OdataID string `json:"@odata.id"`
		} `json:"EntityLink"`
		EntityPciID interface{} `json:"EntityPciId"`
		EntityRole  string      `json:"EntityRole"`
		Identifiers []struct {
			DurableNameFormat string `json:"DurableNameFormat"`
			DurableName       string `json:"DurableName"`
		} `json:"Identifiers"`
		Oem struct {
		} `json:"Oem"`
		PciClassCode      string `json:"PciClassCode"`
		PciFunctionNumber int64  `json:"PciFunctionNumber"`
	} `json:"ConnectedEntities"`
	Description                string `json:"Description"`
	EndpointProtocol           string `json:"EndpointProtocol"`
	HostReservationMemoryBytes int64  `json:"HostReservationMemoryBytes"`
	IPTransportDetails         []struct {
		IPv4Address struct {
			Address       string `json:"Address"`
			AddressOrigin string `json:"AddressOrigin"`
			Gateway       string `json:"Gateway"`
			SubnetMask    string `json:"SubnetMask"`
		} `json:"IPv4Address"`
		IPv6Address struct {
			Address       string `json:"Address"`
			AddressOrigin string `json:"AddressOrigin"`
			AddressState  string `json:"AddressState"`
			PrefixLength  string `json:"PrefixLength"`
		} `json:"IPv6Address"`
		Port              int    `json:"Port"`
		TransportProtocol string `json:"TransportProtocol"`
	} `json:"IPTransportDetails"`
	ID          string `json:"Id"`
	Identifiers []struct {
		DurableName       string `json:"DurableName"`
		DurableNameFormat string `json:"DurableNameFormat"`
	} `json:"Identifiers"`
	Links struct {
		OdataType string `json:"@odata.type"`
		Oem       struct {
			IntelRackScale struct {
				OdataType  string `json:"@odata.type"`
				Interfaces []struct {
					OdataID string `json:"@odata.id"`
				} `json:"Interfaces"`
				Zones []struct {
					OdataID string `json:"@odata.id"`
				} `json:"Zones"`
			} `json:"Intel_RackScale"`
		} `json:"Oem"`
		Ports []interface{} `json:"Ports"`
	} `json:"Links"`
	Name string `json:"Name"`
	Oem  struct {
		IntelRackScale struct {
			OdataType      string `json:"@odata.type"`
			Authentication struct {
				Password string `json:"Password"`
				Username string `json:"Username"`
			} `json:"Authentication"`
		} `json:"Intel_RackScale"`
	} `json:"Oem"`
	PciID      interface{}   `json:"PciId"`
	Redundancy []interface{} `json:"Redundancy"`
	Status     struct {
		Health       string `json:"Health"`
		HealthRollup string `json:"HealthRollup"`
		State        string `json:"State"`
	} `json:"Status"`
}

// GetNQN returns endpoint NQN
func (ep *EndPoint) GetNQN() string {
	for _, identifier := range ep.Identifiers {
		epFormat := identifier.DurableNameFormat
		if strings.ToUpper(epFormat) == "NQN" {
			return identifier.DurableName
		}
	}
	return ""
}
