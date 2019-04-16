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
	// NodesCollectionEntryPoint is a URL path to the RSD Nodes colection
	NodesCollectionEntryPoint = "/redfish/v1/Nodes"
)

// NodesCollection JSON payload structure
type NodesCollection struct {
	OdataContext      string `json:"@odata.context"`
	OdataType         string `json:"@odata.type"`
	Name              string `json:"Name"`
	MembersOdataCount int    `json:"Members@odata.count"`
	Members           []struct {
		OdataID string `json:"@odata.id"`
	} `json:"Members"`
	Actions struct {
		ComposedNodeCollectionAllocate struct {
			Target string `json:"target"`
		} `json:"#ComposedNodeCollection.Allocate"`
	} `json:"Actions"`
	OdataID string `json:"@odata.id"`
}

// Node JSON payload structure
type Node struct {
	OdataContext string `json:"@odata.context"`
	OdataID      string `json:"@odata.id"`
	OdataType    string `json:"@odata.type"`
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	Description  string `json:"Description"`
	UUID         string `json:"UUID"`
	PowerState   string `json:"PowerState"`
	Status       struct {
		State        string `json:"State"`
		Health       string `json:"Health"`
		HealthRollup string `json:"HealthRollup"`
	} `json:"Status"`
	ComposedNodeState string `json:"ComposedNodeState"`
	Boot              struct {
		BootSourceOverrideEnabled                      string   `json:"BootSourceOverrideEnabled"`
		BootSourceOverrideTarget                       string   `json:"BootSourceOverrideTarget"`
		BootSourceOverrideTargetRedfishAllowableValues []string `json:"BootSourceOverrideTarget@Redfish.AllowableValues"`
		BootSourceOverrideMode                         string   `json:"BootSourceOverrideMode"`
		BootSourceOverrideModeRedfishAllowableValues   []string `json:"BootSourceOverrideMode@Redfish.AllowableValues"`
	} `json:"Boot"`
	ClearTPMOnDelete bool `json:"ClearTPMOnDelete"`
	Links            struct {
		ComputerSystem struct {
			OdataID string `json:"@odata.id"`
		} `json:"ComputerSystem"`
		Processors []struct {
			OdataID string `json:"@odata.id"`
		} `json:"Processors"`
		Memory []struct {
			OdataID string `json:"@odata.id"`
		} `json:"Memory"`
		EthernetInterfaces []struct {
			OdataID string `json:"@odata.id"`
		} `json:"EthernetInterfaces"`
		Storage []struct {
			OdataID string `json:"@odata.id"`
		} `json:"Storage"`
		Oem struct {
		} `json:"Oem"`
	} `json:"Links"`
	Actions struct {
		ComposedNodeReset struct {
			Target                          string   `json:"target"`
			ResetTypeRedfishAllowableValues []string `json:"ResetType@Redfish.AllowableValues"`
		} `json:"#ComposedNode.Reset"`
		ComposedNodeAssemble struct {
			Target string `json:"target"`
		} `json:"#ComposedNode.Assemble"`
		ComposedNodeAttachResource struct {
			Target            string `json:"target"`
			RedfishActionInfo struct {
				OdataID string `json:"@odata.id"`
			} `json:"@Redfish.ActionInfo"`
		} `json:"#ComposedNode.AttachResource"`
		ComposedNodeDetachResource struct {
			Target            string `json:"target"`
			RedfishActionInfo struct {
				OdataID string `json:"@odata.id"`
			} `json:"@Redfish.ActionInfo"`
		} `json:"#ComposedNode.DetachResource"`
	} `json:"Actions"`
	Oem struct {
	} `json:"Oem"`
}

// GetMembers returns members of Nodes collection
func (collection *NodesCollection) GetMembers(rsd Transport) ([]*Node, error) {
	var result []*Node
	for _, member := range collection.Members {
		var item Node
		err := rsd.Get(member.OdataID, &item)
		if err != nil {
			return nil, errors.Wrapf(err, "Can't query NodesCollection members %s", member.OdataID)
		}

		result = append(result, &item)
	}
	return result, nil
}

// Action calls node Action
func (node *Node) Action(rsd Transport, odataID, action string) error {
	data := map[string]map[string]string{
		"Resource": {
			"@odata.id": odataID,
		}}

	_, err := rsd.Post(node.OdataID+"/Actions/"+action, data, nil)
	if err != nil {
		return errors.Wrapf(err, "node %s: resource: %s: can't perform action %s", node.OdataID, odataID, action)
	}
	return nil
}

// AttachResource attaches resource to the node
func (node *Node) AttachResource(rsd Transport, odataID string) error {
	return node.Action(rsd, odataID, "ComposedNode.AttachResource")
}

// DetachResource detaches resource from the node
func (node *Node) DetachResource(rsd Transport, odataID string) error {
	return node.Action(rsd, odataID, "ComposedNode.DetachResource")
}
