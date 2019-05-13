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
	"errors"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

func TestNodeGetInfo(t *testing.T) {
	t.Run("Test NodeGetInfo", func(t *testing.T) {
		nodeID := "208909"
		drv := &Driver{RSDNodeID: nodeID}
		got, err := drv.NodeGetInfo(context.Background(), &csi.NodeGetInfoRequest{})
		if err != nil {
			t.Errorf("Driver.NodeGetInfo: unexpected error = %v", err)
			return
		}
		want := &csi.NodeGetInfoResponse{NodeId: nodeID}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("Driver.NodeGetInfo() = %v, want %v", got, want)
		}
	})
}

func TestNodeGetCapabilities(t *testing.T) {
	t.Run("Test NodeGetCapabilities", func(t *testing.T) {
		drv := &Driver{}
		got, err := drv.NodeGetCapabilities(context.Background(), &csi.NodeGetCapabilitiesRequest{})
		if err != nil {
			t.Errorf("Driver.NodeGetCapabilities() unexpected error = %v", err)
			return
		}

		want := &csi.NodeGetCapabilitiesResponse{
			Capabilities: []*csi.NodeServiceCapability{
				&csi.NodeServiceCapability{
					Type: &csi.NodeServiceCapability_Rpc{
						Rpc: &csi.NodeServiceCapability_RPC{
							Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
						},
					},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("Driver.NodeGetCapabilities() = %v, want %v", got, want)
		}
	})
}

// testNVME is a mock nvme structure used to avoid calling nvme tool
type testNVMe struct{}

func (*testNVMe) Connect(transport, traddr, traddrfamily, trsvcid, nqn, hostnqn string) (string, error) {
	return "/dev/nvme1n1", nil
}

func (*testNVMe) Disconnect(device string) error {
	if device == "" {
		return errors.New("device node is empty string")
	}
	return nil
}

// testMounter is a mounter test mock
type testMounter struct{}

func (*testMounter) Mount(source string, target string, fstype string, opts ...string) error {
	return nil
}

func (*testMounter) Unmount(target string) error {
	return nil
}

func (*testMounter) IsMounted(source, target string) (bool, error) {
	return false, nil
}

func (*testMounter) IsFormatted(source string) (bool, error) {
	return false, nil
}

func (*testMounter) Format(source, fsType string) error {
	return nil
}

func TestNodeStageVolume(t *testing.T) {
	tests := []struct {
		name    string
		driver  *Driver
		req     *csi.NodeStageVolumeRequest
		want    *csi.NodeStageVolumeResponse
		wantErr bool
	}{
		{
			name: "correct response",
			driver: &Driver{
				volumes: map[string]*Volume{
					"Vol1": &Volume{
						CSIVolume: &csi.Volume{VolumeId: "1"},
						Name:      "1",
						EndPoint: &endPointInfo{
							transportProtocol: "rdma",
							ipAddress:         "192.168.1.1",
							ipPort:            4420,
							ipAddressFamily:   "IPv4",
							nqn:               "nqn.2000-11.org.nvmexpress:uuid:xxxxx-yyyy-zzzz-0000-ffffffff",
						},
						IsPublished: true,
						IsStaged:    false,
					},
				},
				nvme:    &testNVMe{},
				mounter: &testMounter{},
			},
			req: &csi.NodeStageVolumeRequest{
				VolumeId: "1",
				VolumeCapability: &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{
							FsType:     "ext4",
							MountFlags: []string{},
						},
					},
				},
				StagingTargetPath: "/mnt",
			},
			want:    &csi.NodeStageVolumeResponse{},
			wantErr: false,
		},
		{
			name:    "No Volume ID in the request",
			driver:  &Driver{},
			req:     &csi.NodeStageVolumeRequest{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "No StagingTargetPath in the request",
			driver:  &Driver{},
			req:     &csi.NodeStageVolumeRequest{VolumeId: "1"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "No Volume Capability in the request",
			driver:  &Driver{},
			req:     &csi.NodeStageVolumeRequest{VolumeId: "1", StagingTargetPath: "/mnt"},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "No Volume with requested ID found",
			driver: &Driver{},
			req: &csi.NodeStageVolumeRequest{
				VolumeId:          "1",
				VolumeCapability:  &csi.VolumeCapability{},
				StagingTargetPath: "/mnt",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "volume is not published",
			driver: &Driver{
				volumes: map[string]*Volume{
					"Vol1": &Volume{
						CSIVolume: &csi.Volume{VolumeId: "1"},
					},
				},
			},
			req: &csi.NodeStageVolumeRequest{
				VolumeId: "1",
				VolumeCapability: &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{},
					},
				},
				StagingTargetPath: "/mnt",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.NodeStageVolume(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.NodeStageVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.NodeStageVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeUnstageVolume(t *testing.T) {
	tests := []struct {
		name    string
		driver  *Driver
		req     *csi.NodeUnstageVolumeRequest
		want    *csi.NodeUnstageVolumeResponse
		wantErr bool
	}{
		{
			name: "correct response",
			driver: &Driver{
				volumes: map[string]*Volume{
					"Vol1": &Volume{
						CSIVolume:   &csi.Volume{VolumeId: "1"},
						Name:        "1",
						Device:      "/dev/nvme1n1",
						IsPublished: true,
						IsStaged:    true,
					},
				},
				nvme:    &testNVMe{},
				mounter: &testMounter{},
			},
			req: &csi.NodeUnstageVolumeRequest{
				VolumeId:          "1",
				StagingTargetPath: "/mnt",
				//TargetPath: "/var/docker/pods/pod1/mnt",
			},
			want:    &csi.NodeUnstageVolumeResponse{},
			wantErr: false,
		},
		{
			name: "disconnect fails",
			driver: &Driver{
				volumes: map[string]*Volume{
					"Vol1": &Volume{
						CSIVolume:   &csi.Volume{VolumeId: "1"},
						Name:        "1",
						Device:      "",
						IsPublished: true,
						IsStaged:    true,
					},
				},
				nvme:    &testNVMe{},
				mounter: &testMounter{},
			},
			req: &csi.NodeUnstageVolumeRequest{
				VolumeId:          "1",
				StagingTargetPath: "/mnt",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "No Volume ID in the request",
			driver:  &Driver{},
			req:     &csi.NodeUnstageVolumeRequest{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "No StagingTargetPath in the request",
			driver:  &Driver{},
			req:     &csi.NodeUnstageVolumeRequest{VolumeId: "1"},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "No Volume with requested ID found",
			driver: &Driver{},
			req: &csi.NodeUnstageVolumeRequest{
				VolumeId:          "1",
				StagingTargetPath: "/mnt",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.NodeUnstageVolume(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.NodeUnstageVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.NodeUnstageVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodePublishVolume(t *testing.T) {
	tests := []struct {
		name    string
		driver  *Driver
		req     *csi.NodePublishVolumeRequest
		want    *csi.NodePublishVolumeResponse
		wantErr bool
	}{
		{
			name: "correct response",
			driver: &Driver{
				volumes: map[string]*Volume{
					"Vol1": &Volume{
						CSIVolume: &csi.Volume{VolumeId: "1"},
						Name:      "1",
						EndPoint: &endPointInfo{
							transportProtocol: "rdma",
							ipAddress:         "192.168.1.1",
							ipPort:            4420,
							ipAddressFamily:   "IPv4",
							nqn:               "nqn.2000-11.org.nvmexpress:uuid:xxxxx-yyyy-zzzz-0000-ffffffff",
						},
						IsPublished: false,
						IsStaged:    true,
					},
				},
				mounter: &testMounter{},
			},
			req: &csi.NodePublishVolumeRequest{
				VolumeId: "1",
				VolumeCapability: &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{
							FsType:     "ext4",
							MountFlags: []string{},
						},
					},
				},
				StagingTargetPath: "/mnt",
				TargetPath:        "/var/docker/pods/pod1/mnt",
			},
			want:    &csi.NodePublishVolumeResponse{},
			wantErr: false,
		},
		{
			name:    "No Volume ID in the request",
			driver:  &Driver{},
			req:     &csi.NodePublishVolumeRequest{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "No StagingTargetPath in the request",
			driver:  &Driver{},
			req:     &csi.NodePublishVolumeRequest{VolumeId: "1"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "No TargetPath in the request",
			driver:  &Driver{},
			req:     &csi.NodePublishVolumeRequest{VolumeId: "1", StagingTargetPath: "/mnt"},
			want:    nil,
			wantErr: true,
		},

		{
			name:   "No Volume Capability in the request",
			driver: &Driver{},
			req: &csi.NodePublishVolumeRequest{
				VolumeId:          "1",
				StagingTargetPath: "/mnt",
				TargetPath:        "/var/docker/pod1/mnt",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "No Volume with requested ID found",
			driver: &Driver{},
			req: &csi.NodePublishVolumeRequest{
				VolumeId: "1",
				VolumeCapability: &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{
							FsType:     "ext4",
							MountFlags: []string{},
						},
					},
				},
				StagingTargetPath: "/mnt",
				TargetPath:        "/var/docker/pod1/mnt",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.NodePublishVolume(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.NodePublishVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.NodePublishVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeUnpublishVolume(t *testing.T) {
	tests := []struct {
		name    string
		driver  *Driver
		req     *csi.NodeUnpublishVolumeRequest
		want    *csi.NodeUnpublishVolumeResponse
		wantErr bool
	}{
		{
			name: "correct response",
			driver: &Driver{
				volumes: map[string]*Volume{
					"Vol1": &Volume{
						CSIVolume:   &csi.Volume{VolumeId: "1"},
						Name:        "1",
						IsPublished: true,
						IsStaged:    true,
					},
				},
				mounter: &testMounter{},
			},
			req: &csi.NodeUnpublishVolumeRequest{
				VolumeId:   "1",
				TargetPath: "/var/docker/pods/pod1/mnt",
			},
			want:    &csi.NodeUnpublishVolumeResponse{},
			wantErr: false,
		},
		{
			name:    "No Volume ID in the request",
			driver:  &Driver{},
			req:     &csi.NodeUnpublishVolumeRequest{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "No TargetPath in the request",
			driver:  &Driver{},
			req:     &csi.NodeUnpublishVolumeRequest{VolumeId: "1"},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "No Volume with requested ID found",
			driver: &Driver{},
			req: &csi.NodeUnpublishVolumeRequest{
				VolumeId:   "1",
				TargetPath: "/var/docker/pod1/mnt",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.driver.NodeUnpublishVolume(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Driver.NodeUnpublishVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.NodeUnpublishVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}
