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

// ComputerSystem JSON payload structure
type ComputerSystem struct {
	OdataContext string `json:"@odata.context"`
	OdataID      string `json:"@odata.id"`
	OdataType    string `json:"@odata.type"`
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	Description  string `json:"Description"`
	SystemType   string `json:"SystemType"`
	AssetTag     string `json:"AssetTag"`
	Manufacturer string `json:"Manufacturer"`
	Model        string `json:"Model"`
	SKU          string `json:"SKU"`
	SerialNumber string `json:"SerialNumber"`
	PartNumber   string `json:"PartNumber"`
	UUID         string `json:"UUID"`
	HostName     string `json:"HostName"`
	Status       struct {
		State        string `json:"State"`
		Health       string `json:"Health"`
		HealthRollup string `json:"HealthRollup"`
	} `json:"Status"`
	IndicatorLED   string      `json:"IndicatorLED"`
	PowerState     string      `json:"PowerState"`
	BiosVersion    interface{} `json:"BiosVersion"`
	HostedServices struct {
		StorageServices interface{} `json:"StorageServices"`
	} `json:"HostedServices"`
	HostingRoles []interface{} `json:"HostingRoles"`
	Boot         struct {
		OdataType                                      string   `json:"@odata.type"`
		BootSourceOverrideEnabled                      string   `json:"BootSourceOverrideEnabled"`
		BootSourceOverrideTarget                       string   `json:"BootSourceOverrideTarget"`
		BootSourceOverrideTargetRedfishAllowableValues []string `json:"BootSourceOverrideTarget@Redfish.AllowableValues"`
		BootSourceOverrideMode                         string   `json:"BootSourceOverrideMode"`
		BootSourceOverrideModeRedfishAllowableValues   []string `json:"BootSourceOverrideMode@Redfish.AllowableValues"`
	} `json:"Boot"`
	ProcessorSummary struct {
		Count  int    `json:"Count"`
		Model  string `json:"Model"`
		Status struct {
			State        string `json:"State"`
			Health       string `json:"Health"`
			HealthRollup string `json:"HealthRollup"`
		} `json:"Status"`
	} `json:"ProcessorSummary"`
	MemorySummary struct {
		TotalSystemMemoryGiB float64 `json:"TotalSystemMemoryGiB"`
		Status               struct {
			State        string `json:"State"`
			Health       string `json:"Health"`
			HealthRollup string `json:"HealthRollup"`
		} `json:"Status"`
	} `json:"MemorySummary"`
	Processors struct {
		OdataID string `json:"@odata.id"`
	} `json:"Processors"`
	EthernetInterfaces struct {
		OdataID string `json:"@odata.id"`
	} `json:"EthernetInterfaces"`
	NetworkInterfaces struct {
		OdataID string `json:"@odata.id"`
	} `json:"NetworkInterfaces"`
	Storage struct {
		OdataID string `json:"@odata.id"`
	} `json:"Storage"`
	Memory struct {
		OdataID string `json:"@odata.id"`
	} `json:"Memory"`
	PCIeDevices    []interface{} `json:"PCIeDevices"`
	PCIeFunctions  []interface{} `json:"PCIeFunctions"`
	TrustedModules []interface{} `json:"TrustedModules"`
	Links          struct {
		OdataType string            `json:"@odata.type"`
		Chassis   []interface{}     `json:"Chassis"`
		Endpoints []endPointOdataID `json:"Endpoints"`
		ManagedBy []struct {
			OdataID string `json:"@odata.id"`
		} `json:"ManagedBy"`
		Oem struct {
		} `json:"Oem"`
	} `json:"Links"`
	Actions struct {
		Oem struct {
			IntelOemChangeTPMState struct {
				Target                              string        `json:"target"`
				InterfaceTypeRedfishAllowableValues []interface{} `json:"InterfaceType@Redfish.AllowableValues"`
			} `json:"#Intel.Oem.ChangeTPMState"`
		} `json:"Oem"`
		ComputerSystemReset struct {
			Target                          string   `json:"target"`
			ResetTypeRedfishAllowableValues []string `json:"ResetType@Redfish.AllowableValues"`
		} `json:"#ComputerSystem.Reset"`
	} `json:"Actions"`
	Oem struct {
		IntelRackScale struct {
			OdataType                         string        `json:"@odata.type"`
			PciDevices                        []interface{} `json:"PciDevices"`
			PCIeConnectionID                  []string      `json:"PCIeConnectionId"`
			ProcessorSockets                  int           `json:"ProcessorSockets"`
			MemorySockets                     int           `json:"MemorySockets"`
			DiscoveryState                    string        `json:"DiscoveryState"`
			UserModeEnabled                   bool          `json:"UserModeEnabled"`
			TrustedExecutionTechnologyEnabled bool          `json:"TrustedExecutionTechnologyEnabled"`
			Metrics                           struct {
				OdataID string `json:"@odata.id"`
			} `json:"Metrics"`
		} `json:"Intel_RackScale"`
	} `json:"Oem"`
}

// GetEndPoints returns List of EndPoints associated with a Volume
func (cs *ComputerSystem) GetEndPoints(rsd Transport) ([]*EndPoint, error) {
	return GetEndPoints(rsd, cs.Links.Endpoints)
}
