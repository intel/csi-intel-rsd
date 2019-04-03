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
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/intel/csi-intel-rsd/pkg/rsd"
)

func TestControllerGetCapabilities(t *testing.T) {
	t.Run("Test ControllerGetCapabilities", func(t *testing.T) {
		drv := &Driver{}
		got, err := drv.ControllerGetCapabilities(context.Background(), &csi.ControllerGetCapabilitiesRequest{})
		if err != nil {
			t.Errorf("Driver.ControllerGetCapabilities() unexpected error = %v", err)
			return
		}

		want := &csi.ControllerGetCapabilitiesResponse{
			Capabilities: []*csi.ControllerServiceCapability{
				newCap(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME),
				newCap(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME),
				newCap(csi.ControllerServiceCapability_RPC_LIST_VOLUMES),
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("Driver.ControllerGetCapabilities() = %v, want %v", got, want)
		}
	})
}

func TestListVolumes(t *testing.T) {
	tests := []struct {
		name    string
		driver  *Driver
		want    *csi.ListVolumesResponse
		wantErr bool
	}{
		{
			name: "Correct response",
			driver: &Driver{
				volumes: map[string]*Volume{
					"Vol1": &Volume{
						CSIVolume: &csi.Volume{CapacityBytes: 100},
					},
					"Vol2": &Volume{
						CSIVolume: &csi.Volume{CapacityBytes: 200},
					},
				},
			},
			want: &csi.ListVolumesResponse{
				Entries: []*csi.ListVolumesResponse_Entry{
					&csi.ListVolumesResponse_Entry{
						Volume: &csi.Volume{CapacityBytes: 100},
					},
					&csi.ListVolumesResponse_Entry{
						Volume: &csi.Volume{CapacityBytes: 200},
					},
				},
				NextToken: "",
			},
			wantErr: false,
		},
		{
			name:    "Empty response",
			driver:  &Driver{},
			want:    &csi.ListVolumesResponse{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.ListVolumes(context.Background(), &csi.ListVolumesRequest{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.ListVolumes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.ListVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateVolumeCapabilities(t *testing.T) {
	tests := []struct {
		name    string
		driver  *Driver
		req     *csi.ValidateVolumeCapabilitiesRequest
		want    *csi.ValidateVolumeCapabilitiesResponse
		wantErr bool
	}{
		{
			name: "Correct response",
			driver: &Driver{
				volumes: map[string]*Volume{
					"Vol1": &Volume{
						CSIVolume: &csi.Volume{VolumeId: "Vol1"},
					},
				},
			},
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeId: "Vol1",
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			},
			want: &csi.ValidateVolumeCapabilitiesResponse{
				Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
					VolumeCapabilities: []*csi.VolumeCapability{
						{
							AccessMode: &csi.VolumeCapability_AccessMode{
								Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "No VolumeID in the request",
			driver:  &Driver{},
			req:     &csi.ValidateVolumeCapabilitiesRequest{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "No Volume Capabilities in the request",
			driver:  &Driver{},
			req:     &csi.ValidateVolumeCapabilitiesRequest{VolumeId: "Vol1"},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "Volume not found",
			driver: &Driver{},
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeId: "Vol1",
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Unsupported access mode",
			driver: &Driver{
				volumes: map[string]*Volume{
					"Vol1": &Volume{
						CSIVolume: &csi.Volume{VolumeId: "Vol1"},
					},
				},
			},
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeId: "Vol1",
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
			},
			want: &csi.ValidateVolumeCapabilitiesResponse{
				Confirmed: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.ValidateVolumeCapabilities(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.ValidateVolumeCapabilities() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.ValidateVolumeCapabilities() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCapabilities(t *testing.T) {
	type args struct {
		caps []*csi.VolumeCapability
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateCapabilities(tt.args.caps); got != tt.want {
				t.Errorf("validateCapabilities() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestClient is a mock that implements rsd.Transport interface
type TestClient struct {
	results map[string]string
}

// Get gets json string from TestClient.results and decodes into the result
func (client *TestClient) Get(entrypoint string, result interface{}) error {
	res, ok := client.results[entrypoint]
	if !ok {
		return fmt.Errorf("Unsupported entry point: %s", entrypoint)
	}
	err := json.NewDecoder(strings.NewReader(res)).Decode(result)
	if err != nil {
		return fmt.Errorf("Can't decode json string '%s', error: %v", res, err)
	}
	return nil
}

// Post returns correct location
func (client *TestClient) Post(entrypoint string, data interface{}, result interface{}) (*http.Header, error) {
	return &http.Header{"Location": []string{"/redfish/v1/StorageServices/1/Volumes/1"}}, nil
}

// Delete does nothing
func (client *TestClient) Delete(entrypoint string, data interface{}, result interface{}) (*http.Header, error) {
	return nil, nil
}

func TestCreateVolume(t *testing.T) {
	tests := []struct {
		name    string
		driver  *Driver
		req     *csi.CreateVolumeRequest
		want    *csi.CreateVolumeResponse
		wantErr bool
	}{
		{
			name: "Create new volume",
			driver: &Driver{
				rsdClient: &TestClient{
					results: map[string]string{
						"/redfish/v1/StorageServices":             "{\"Members\": [{\"@odata.id\": \"/redfish/v1/StorageServices/1\"}]}",
						"/redfish/v1/StorageServices/1":           "{\"Volumes\": {\"@odata.id\": \"/redfish/v1/StorageServices/1/Volumes\"}}",
						"/redfish/v1/StorageServices/1/Volumes":   "{\"Members\": [{\"@odata.id\": \"/redfish/v1/StorageServices/1/Volumes/1\"}]}",
						"/redfish/v1/StorageServices/1/Volumes/1": "{\"Id\": 1, \"CapacityBytes\": \"100\"}",
					},
				},
				volumes: map[string]*Volume{},
			},
			req: &csi.CreateVolumeRequest{
				Name: "CSI-generated",
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
				CapacityRange: &csi.CapacityRange{
					RequiredBytes: 50,
					LimitBytes:    100,
				},
			},
			want: &csi.CreateVolumeResponse{
				Volume: &csi.Volume{
					VolumeId:      "1",
					CapacityBytes: 100,
					VolumeContext: map[string]string{
						"name": "CSI-generated",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Existing volume",
			driver: &Driver{
				rsdClient: &TestClient{
					results: map[string]string{
						"/redfish/v1/StorageServices":             "{\"Members\": [{\"@odata.id\": \"/redfish/v1/StorageServices/1\"}]}",
						"/redfish/v1/StorageServices/1":           "{\"Volumes\": {\"@odata.id\": \"/redfish/v1/StorageServices/1/Volumes\"}}",
						"/redfish/v1/StorageServices/1/Volumes":   "{\"Members\": [{\"@odata.id\": \"/redfish/v1/StorageServices/1/Volumes/1\"}]}",
						"/redfish/v1/StorageServices/1/Volumes/1": "{\"Id\": 1, \"CapacityBytes\": \"100\"}",
					},
				},
				volumes: map[string]*Volume{
					"CSI-generated": &Volume{
						CSIVolume: &csi.Volume{
							VolumeId:      "1",
							VolumeContext: map[string]string{"name": "CSI-generated"},
							CapacityBytes: 100,
						},
					},
				},
			},
			req: &csi.CreateVolumeRequest{
				Name: "CSI-generated",
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
				CapacityRange: &csi.CapacityRange{
					RequiredBytes: 50,
					LimitBytes:    100,
				},
			},
			want: &csi.CreateVolumeResponse{
				Volume: &csi.Volume{
					VolumeId:      "1",
					CapacityBytes: 100,
					VolumeContext: map[string]string{
						"name": "CSI-generated",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "missing Volume name",
			driver:  &Driver{},
			req:     &csi.CreateVolumeRequest{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing Volume Capabilities",
			driver:  &Driver{},
			req:     &csi.CreateVolumeRequest{Name: "CSI-generated"},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "Invalid Volume Capabilities",
			driver: &Driver{},
			req: &csi.CreateVolumeRequest{
				Name: "CSI-generated",
				VolumeCapabilities: []*csi.VolumeCapability{
					&csi.VolumeCapability{
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.CreateVolume(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.CreateVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.CreateVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteVolume(t *testing.T) {
	tests := []struct {
		name    string
		driver  *Driver
		req     *csi.DeleteVolumeRequest
		want    *csi.DeleteVolumeResponse
		wantErr bool
	}{
		{
			name: "Delete existing volume",
			driver: &Driver{
				rsdClient: &TestClient{},
				volumes: map[string]*Volume{
					"CSI-generated": &Volume{
						RSDVolume: &rsd.Volume{},
						CSIVolume: &csi.Volume{
							VolumeId:      "1",
							VolumeContext: map[string]string{"name": "CSI-generated"},
							CapacityBytes: 100,
						},
					},
				},
			},
			req:     &csi.DeleteVolumeRequest{VolumeId: "1"},
			want:    &csi.DeleteVolumeResponse{},
			wantErr: false,
		},
		{
			name:    "missing Volume Id",
			driver:  &Driver{},
			req:     &csi.DeleteVolumeRequest{},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.DeleteVolume(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.DeleteVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.DeleteVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPublishVolume(t *testing.T) {
	testClient := &TestClient{
		results: map[string]string{
			"/redfish/v1/StorageServices": `{
				"Members": [
					{
						"@odata.id": "/redfish/v1/StorageServices/1"
					}
				]
			}`,
			"/redfish/v1/StorageServices/1": `{
				"Volumes": {
					"@odata.id": "/redfish/v1/StorageServices/1/Volumes"
				}
			}`,
			"/redfish/v1/StorageServices/1/Volumes": `{
				"Members": [
					{
						"@odata.id": "/redfish/v1/StorageServices/1/Volumes/1"
					}
				]
			}`,
			"/redfish/v1/StorageServices/1/Volumes/1": `{
				"Id": 1,
				"CapacityBytes": "100",
				"Links": {
					"Oem": {
						"Intel_RackScale": {
							"Endpoints": [
								{
									"@odata.id": "/redfish/v1/Fabrics/1/Endpoints/nqn.1"
								}
							]
						}
					}
				}
			}`,
			"/redfish/v1/Nodes": `{
				"Members": [
					{
						"@odata.id": "/redfish/v1/Nodes/1"
					}
				]
			}`,
			"/redfish/v1/Nodes/1": `{
				"@odata.id": "/redfish/v1/Nodes/1",
				"ID": "1"
			}`,
			"/redfish/v1/Fabrics/1/Endpoints/nqn.1": `{
				"IPTransportDetails": [
					{
						"IPv4Address": {
							"Address": "192.168.1.1"
						},
						"IPv6Address": {
							"Address": null
						},
						"Port": 4420,
						"TransportProtocol": "RoCEv2"
					}
				],
				"Id": "nqn.1",
				"Identifiers": [
					{
						"DurableName": "nqn.1",
						"DurableNameFormat": "NQN"
					}
				]
			}`,
		},
	}

	rsdVolume, err := rsd.GetVolume(testClient, 0, 1)
	if err != nil {
		t.Fatalf("can't get volume id 1: %v", err)
	}

	tests := []struct {
		name    string
		driver  *Driver
		req     *csi.ControllerPublishVolumeRequest
		want    *csi.ControllerPublishVolumeResponse
		wantErr bool
	}{
		{
			name: "Publish existing volume",
			driver: &Driver{
				rsdClient: testClient,
				volumes: map[string]*Volume{
					"CSI-generated": &Volume{
						RSDVolume: rsdVolume,
						CSIVolume: &csi.Volume{
							VolumeId:      "1",
							VolumeContext: map[string]string{"name": "CSI-generated"},
							CapacityBytes: 100,
						},
					},
				},
			},
			req: &csi.ControllerPublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "1",
				VolumeCapability: &csi.VolumeCapability{
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			want: &csi.ControllerPublishVolumeResponse{
				PublishContext: map[string]string{
					PublishInfoVolumeName: "CSI-generated",
				},
			},
			wantErr: false,
		},
		{
			name: "node doesn't exist",
			driver: &Driver{
				rsdClient: &TestClient{
					results: map[string]string{
						"/redfish/v1/Nodes": "{\"Members\": []}",
					},
				},
				volumes: map[string]*Volume{
					"CSI-generated": &Volume{
						RSDVolume: &rsd.Volume{},
						CSIVolume: &csi.Volume{
							VolumeId:      "1",
							VolumeContext: map[string]string{"name": "CSI-generated"},
							CapacityBytes: 100,
						},
					},
				},
			},
			req: &csi.ControllerPublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "1",
				VolumeCapability: &csi.VolumeCapability{
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing Volume Id",
			driver:  &Driver{},
			req:     &csi.ControllerPublishVolumeRequest{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing Node Id",
			driver:  &Driver{},
			req:     &csi.ControllerPublishVolumeRequest{VolumeId: "1"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing Volume Capabilities",
			driver:  &Driver{},
			req:     &csi.ControllerPublishVolumeRequest{VolumeId: "1", NodeId: "1"},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "volume doesn't exist",
			driver: &Driver{},
			req: &csi.ControllerPublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "1",
				VolumeCapability: &csi.VolumeCapability{
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.ControllerPublishVolume(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.PublishVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.PublishVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnpublishVolume(t *testing.T) {
	tests := []struct {
		name    string
		driver  *Driver
		req     *csi.ControllerUnpublishVolumeRequest
		want    *csi.ControllerUnpublishVolumeResponse
		wantErr bool
	}{
		{
			name: "Unpublish existing volume",
			driver: &Driver{
				rsdClient: &TestClient{
					results: map[string]string{
						"/redfish/v1/Nodes":   "{\"Members\": [{\"@odata.id\": \"/redfish/v1/Nodes/1\"}]}",
						"/redfish/v1/Nodes/1": "{\"@odata.id\": \"/redfish/v1/Nodes/1\", \"ID\": \"1\"}",
					},
				},

				volumes: map[string]*Volume{
					"CSI-generated": &Volume{
						RSDVolume: &rsd.Volume{},
						CSIVolume: &csi.Volume{
							VolumeId:      "1",
							VolumeContext: map[string]string{"name": "CSI-generated"},
							CapacityBytes: 100,
						},
						IsPublished: true,
					},
				},
			},
			req: &csi.ControllerUnpublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "1",
			},
			want:    &csi.ControllerUnpublishVolumeResponse{},
			wantErr: false,
		},
		{
			name: "node doesn't exist",
			driver: &Driver{
				rsdClient: &TestClient{
					results: map[string]string{
						"/redfish/v1/Nodes": "{\"Members\": []}",
					},
				},
				volumes: map[string]*Volume{
					"CSI-generated": &Volume{
						RSDVolume: &rsd.Volume{},
						CSIVolume: &csi.Volume{
							VolumeId:      "1",
							VolumeContext: map[string]string{"name": "CSI-generated"},
							CapacityBytes: 100,
						},
						IsPublished: true,
					},
				},
			},
			req: &csi.ControllerUnpublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "1",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing Volume Id",
			driver:  &Driver{},
			req:     &csi.ControllerUnpublishVolumeRequest{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing Node Id",
			driver:  &Driver{},
			req:     &csi.ControllerUnpublishVolumeRequest{VolumeId: "1"},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "volume doesn't exist",
			driver: &Driver{},
			req: &csi.ControllerUnpublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "1",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.ControllerUnpublishVolume(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.UnpublishVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.UnpublishVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}
