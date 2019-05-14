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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NodeGetInfo returns the supported capabilities of the node server.
// This is used so the CO knows where to place the workload. The result of this
// function will be used by the CO in ControllerPublishVolume.
func (drv *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	log.Printf("NodeGetInfo request: %v", req)

	resp := &csi.NodeGetInfoResponse{NodeId: drv.RSDNodeID}

	log.Printf("NodeGetInfo response: %v", resp)
	return resp, nil
}

// NodeGetCapabilities returns the supported capabilities of the node server
func (drv *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	log.Printf("NodeGetCapabilities request: %v", req)

	resp := &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			&csi.NodeServiceCapability{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
			&csi.NodeServiceCapability{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
					},
				},
			},
		},
	}

	log.Printf("NodeGetCapabilities response: %v", resp)
	return resp, nil
}

// NodeGetVolumeStats returns the volume capacity statistics available for the given volume.
func (drv *Driver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	log.Printf("NodeGetVolumeStats request: %v", req)
	if req == nil || req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID can't be empty")
	}

	if req.VolumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume Path cannot be empty")
	}

	// Check if volume path is an absolute path
	//if !filepath.IsAbs(req.VolumePath) {
	//	return nil, status.Error(codes.InvalidArgument, "Volume Path must be absolute")
	//}

	// Check if volume path exists
	_, vol := drv.findVolByID(req.VolumeId)
	if vol == nil {
		return nil, status.Errorf(codes.NotFound, "Volume Id '%s' not found", req.VolumeId)
	}

	// Check if volume path is either stagingtarget or target path
	_, exists := vol.TargetPaths[req.VolumePath]
	if !exists && req.VolumePath != vol.StagingTargetPath {
		return nil, status.Errorf(codes.NotFound, "Path '%s' is neither a staging target path nor target path for the volume '%s'", req.VolumePath, req.VolumeId)
	}

	resp := &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				Total: vol.RSDVolume.CapacityBytes,
				Unit:  csi.VolumeUsage_BYTES,
			},
		},
	}

	log.Printf("NodeGetVolumeStats response: %v", resp)
	return resp, nil
}

// getFstype returns FS type considering its default value "ext4"
func getFsType(fsType string) string {
	if fsType != "" {
		return fsType
	}
	return "ext4"
}

// NodeStageVolume mounts the volume to a staging path on the node. This is
// called by the CO before NodePublishVolume and is used to temporary mount the
// volume to a staging path. Once mounted, NodePublishVolume will make sure to
// bindmount it to the appropriate path
func (drv *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	log.Printf("NodeStageVolume request: %v", req)

	if req == nil || req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume: Volume ID can't be empty")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume: Staging Target Path is missing")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume: Volume capability is missing")
	}

	// lock driver volumes to satisfy idepotency requirements
	drv.volumesRWL.Lock()
	defer drv.volumesRWL.Unlock()

	// Check if the volume exists
	name, vol := drv.findVolByID(req.VolumeId)
	if name == "" {
		return nil, status.Errorf(codes.NotFound, "NodeStageVolume: No volume with id '%s' found", req.VolumeId)
	}

	mnt := req.VolumeCapability.GetMount()

	err := drv.nodeStageVolume(vol, getFsType(mnt.FsType), req.StagingTargetPath, mnt.MountFlags)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "NodeStageVolume: error staging volume %s(%s) on the path %s: %v", name, req.VolumeId, req.StagingTargetPath, err)
	}

	log.Printf("NodeStageVolume: volume %s(%s) has been staged on the path %s", name, req.VolumeId, req.StagingTargetPath)
	return &csi.NodeStageVolumeResponse{}, nil
}

// NodeUnstageVolume unstages the volume from the staging path
func (drv *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	log.Printf("NodeUnstageVolume request: %v", req)

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume: Volume ID is missing")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume: Staging Target Path is missing")
	}

	// lock driver volumes to satisfy idepotency requirements
	drv.volumesRWL.Lock()
	defer drv.volumesRWL.Unlock()

	// Check if the volume exists
	name, vol := drv.findVolByID(req.VolumeId)
	if name == "" {
		return nil, status.Errorf(codes.NotFound, "NodeUnstageVolume: No volume with id '%s' found", req.VolumeId)
	}

	err := drv.nodeUnstageVolume(vol, req.StagingTargetPath)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "NodeUnstageVolume: error unstaging volume %s(%s) from the path %s: %v", name, req.VolumeId, req.StagingTargetPath, err)
	}

	log.Printf("NodeUnstageVolume: volume %s(%s) has been unstaged from the path %s", name, req.VolumeId, req.StagingTargetPath)
	return &csi.NodeUnstageVolumeResponse{}, nil
}

// NodePublishVolume mounts the volume mounted to the staging path to the target path
func (drv *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	log.Printf("NodePublishVolume request: %v", req)

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: Volume ID is missing")
	}

	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: Staging Target Path is missing")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: Target Path is missing")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: Volume Capability is missing")
	}

	mnt := req.VolumeCapability.GetMount()
	options := mnt.MountFlags

	options = append(options, "bind")
	if req.Readonly {
		options = append(options, "ro")
	}

	// lock driver volumes to satisfy idepotency requirements
	drv.volumesRWL.Lock()
	defer drv.volumesRWL.Unlock()

	// Check if the volume exists
	name, vol := drv.findVolByID(req.VolumeId)
	if name == "" {
		return nil, status.Errorf(codes.NotFound, "NodePublishVolume: No volume with id '%s' found", req.VolumeId)
	}

	err := drv.nodePublishVolume(vol, getFsType(mnt.FsType), req.StagingTargetPath, req.TargetPath, options)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "NodePublishVolume: error publishing volume id %s on %s: %v", req.VolumeId, req.StagingTargetPath, err)
	}

	log.Printf("NodePublishVolume: volume id %s has been published on the path %s", req.VolumeId, req.StagingTargetPath)
	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmounts the volume from the target path
func (drv *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	log.Printf("NodeUnpublishVolume request: %v", req)

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume: Volume ID is missing")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume: Target Path is missing")
	}

	// lock driver volumes to satisfy idepotency requirements
	drv.volumesRWL.Lock()
	defer drv.volumesRWL.Unlock()

	// Check if the volume exists
	name, vol := drv.findVolByID(req.VolumeId)
	if name == "" {
		return nil, status.Errorf(codes.NotFound, "NodeUpublishVolume: No volume with id '%s' found", req.VolumeId)
	}

	err := drv.nodeUnpublishVolume(vol, req.TargetPath)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "NodeUnpublishVolume: error unpublishing volume id %s from the path %s: %v", req.VolumeId, req.TargetPath, err)
	}

	log.Printf("NodeUnpublishVolume: volume id %s has been unpublished from the target path %s", req.VolumeId, req.TargetPath)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}
