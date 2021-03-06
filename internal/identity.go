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

package csirsd

import (
	"context"
	"log"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/ptypes/wrappers"
)

// GetPluginInfo returns metadata of the plugin
func (drv *Driver) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	log.Printf("GetPluginInfo request: %v", req)

	resp := &csi.GetPluginInfoResponse{
		Name:          DriverName,
		VendorVersion: DriverVersion,
	}

	log.Printf("GetPluginInfo response: %v", resp)
	return resp, nil
}

// GetPluginCapabilities returns available capabilities of the plugin
func (drv *Driver) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	log.Printf("GetPluginCapabilities request: %v", req)

	resp := &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}

	log.Printf("GetPluginCapabilities response: %v", resp)
	return resp, nil
}

// Probe returns the health and readiness of the plugin
func (drv *Driver) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	log.Printf("Probe request: %v", req)

	drv.readyMu.Lock()
	defer drv.readyMu.Unlock()

	resp := &csi.ProbeResponse{
		Ready: &wrappers.BoolValue{
			Value: drv.ready,
		},
	}

	log.Printf("Probe response: %v", resp)
	return resp, nil
}
