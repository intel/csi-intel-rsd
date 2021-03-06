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
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Size constants (Kilobytes, Megabytes, etc)
const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const (
	defaultVolumeCapacity int64 = 16 * MB
)

func newCap(cap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
	return &csi.ControllerServiceCapability{
		Type: &csi.ControllerServiceCapability_Rpc{
			Rpc: &csi.ControllerServiceCapability_RPC{
				Type: cap,
			},
		},
	}
}

// ControllerGetCapabilities returns the capabilities of the controller service.
func (drv *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	log.Printf("ControllerGetCapabilities request: %v", req)
	var caps []*csi.ControllerServiceCapability
	for _, cap := range []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_GET_CAPACITY,
	} {
		caps = append(caps, newCap(cap))
	}

	resp := &csi.ControllerGetCapabilitiesResponse{
		Capabilities: caps,
	}

	log.Printf("ControllerGetCapabilities response: %v", resp)

	return resp, nil
}

// ListVolumes returns a list of available volumes created by the driver
func (drv *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	log.Printf("ListVolumes request: %v", req)

	var startingToken int
	var err error
	if req.StartingToken != "" {
		startingToken, err = strconv.Atoi(req.StartingToken)
		if err != nil {
			return nil, status.Errorf(codes.Aborted, "can't convert startingToken %s into int32: %v", req.StartingToken, err)
		}
	}

	// lock driver volumes to satisfy idepotency requirements
	drv.volumesRWL.Lock()
	defer drv.volumesRWL.Unlock()

	volumes := drv.listCSIVolumes()
	numVols := len(volumes)
	if startingToken > numVols {
		return nil, status.Errorf(codes.Aborted, "startingToken %d is greater than amount of volumes %d", startingToken, numVols)
	}

	numEntries := numVols - startingToken
	var nextToken string
	if req.MaxEntries > 0 && req.MaxEntries < int32(numEntries) {
		numEntries = int(req.MaxEntries)
		nextToken = strconv.Itoa(startingToken + numEntries)
	}

	var entries []*csi.ListVolumesResponse_Entry
	for i, volume := range volumes {
		if i >= startingToken {
			entries = append(entries,
				&csi.ListVolumesResponse_Entry{
					Volume: volume,
				})
			if len(entries) == numEntries {
				break
			}
		}
	}

	resp := &csi.ListVolumesResponse{Entries: entries, NextToken: nextToken}

	log.Printf("ListVolumes response: %v", resp)
	return resp, nil
}

// ValidateVolumeCapabilities checks if requested volume capabilities are supported
func (drv *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	log.Printf("ValidateVolumeCapabilities request: %v", req)

	if req == nil || req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID can't be empty")
	}

	if req.VolumeCapabilities == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities must be provided")
	}

	// Check if volume exists
	_, vol := drv.findVolByID(req.VolumeId)
	if vol == nil {
		return nil, status.Errorf(codes.NotFound, "Volume Id '%s' not found", req.VolumeId)
	}

	resp := &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeContext:      req.VolumeContext,
			VolumeCapabilities: req.VolumeCapabilities,
			Parameters:         req.Parameters,
		},
	}

	for _, cap := range req.VolumeCapabilities {
		// Only confirm requests for supported mode
		if cap.AccessMode != nil && cap.AccessMode.Mode != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
			resp.Confirmed = nil
			return resp, status.Errorf(codes.InvalidArgument, "Unsupported Access Mode: %v", cap.AccessMode)
		}
	}

	log.Printf("ValidateVolumeCapabilities response: %v", resp)
	return resp, nil
}

// validateCapabilities validates the requested capabilities.
func validateCapabilities(caps []*csi.VolumeCapability) bool {
	vcaps := []*csi.VolumeCapability_AccessMode{&csi.VolumeCapability_AccessMode{
		Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	}}

	hasSupport := func(mode csi.VolumeCapability_AccessMode_Mode) bool {
		for _, m := range vcaps {
			if mode == m.Mode {
				return true
			}
		}
		return false
	}

	for _, cap := range caps {
		if !hasSupport(cap.AccessMode.Mode) {
			return false
		}
	}

	return true
}

func getRequiredCapacity(req *csi.CreateVolumeRequest) int64 {
	requiredCapacity := defaultVolumeCapacity
	if capRange := req.CapacityRange; capRange != nil {
		if requiredBytes := capRange.GetRequiredBytes(); requiredBytes > 0 {
			requiredCapacity = requiredBytes
		}
		if limitBytes := capRange.GetLimitBytes(); limitBytes > 0 {
			requiredCapacity = limitBytes
		}
	}
	return requiredCapacity
}

// CreateVolume creates new RSD Volume
func (drv *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	log.Printf("CreateVolume request: %v", req)

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume Name can't be empty")
	}

	if req.VolumeCapabilities == nil || len(req.VolumeCapabilities) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Volume %s: capabilities are missing", req.Name)
	}

	if !validateCapabilities(req.VolumeCapabilities) {
		return nil, status.Errorf(codes.InvalidArgument, "Volume %s: invalid volume capabilities requested. Only SINGLE_NODE_WRITER is supported", req.Name)
	}

	// get required capacity
	requiredCapacity := getRequiredCapacity(req)

	// lock driver volumes to satisfy idepotency requirements
	drv.volumesRWL.Lock()
	defer drv.volumesRWL.Unlock()

	// Check if the volume already exists.
	if vol := drv.findCSIVolumeByName(req.Name); vol != nil {
		// Check if existing volume's capacity satisfies request
		capacityBytes := vol.GetCapacityBytes()
		if capacityBytes < requiredCapacity {
			return nil, status.Errorf(codes.AlreadyExists, "Volume %s has smaller size(%d) than required(%d)", req.Name, capacityBytes, requiredCapacity)
		}
		return &csi.CreateVolumeResponse{Volume: vol}, nil
	}

	// Volume doesn't exist - create new one
	vol, err := drv.newVolume(req.Name, requiredCapacity)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &csi.CreateVolumeResponse{Volume: vol}

	log.Printf("CreateVolume response: %v", resp)
	return resp, nil
}

// DeleteVolume deletes existing RSD Volume
func (drv *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	log.Printf("DeleteVolume request: %v", req)

	//  If the volume is not specified, return error
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID is missing")
	}

	err := drv.deleteVolume(req.VolumeId)
	if err != nil {
		return nil, err
	}

	log.Printf("DeleteVolume: volume %s has been deleted", req.VolumeId)
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume attaches the given volume to the node
func (drv *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	log.Printf("ControllerPublishVolume request: %v", req)

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID is missing")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Node ID is missing")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability is missing")
	}

	// lock driver volumes to satisfy idepotency requirements
	drv.volumesRWL.Lock()
	defer drv.volumesRWL.Unlock()

	// Check if the volume exists
	name, vol := drv.findVolByID(req.VolumeId)
	if name == "" {
		return nil, status.Errorf(codes.NotFound, "No volume with id '%s' found", req.VolumeId)
	}

	// Check if node ID is correct
	if drv.RSDNodeID != req.NodeId {
		return nil, status.Errorf(codes.NotFound, "No node with id '%s' found", req.NodeId)
	}

	err := drv.publishVolume(vol, req.NodeId)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "error attaching volume %s(%s) to the node %s: %v", name, req.VolumeId, req.NodeId, err)
	}

	log.Printf("volume %s(%s) has been attached to the node %s", name, req.VolumeId, req.NodeId)

	resp := &csi.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{
			PublishInfoVolumeName: name,
		},
	}

	log.Printf("ControllerPublishVolume response: %v", resp)
	return resp, nil
}

// ControllerUnpublishVolume deattaches the given volume from the node
func (drv *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	log.Printf("ControllerUnpublishVolume request: %v", req)

	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID is missing")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Node ID is missing")
	}

	// lock driver volumes to satisfy idepotency requirements
	drv.volumesRWL.Lock()
	defer drv.volumesRWL.Unlock()

	// Check if the volume exists
	name, vol := drv.findVolByID(req.VolumeId)
	if name == "" {
		return nil, status.Errorf(codes.NotFound, "No volume with id '%s' found", req.VolumeId)
	}

	// Check if node ID is correct
	if drv.RSDNodeID != req.NodeId {
		return nil, status.Errorf(codes.NotFound, "No node with id '%s' found", req.NodeId)
	}

	err := drv.unpublishVolume(vol, req.NodeId)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "error detaching volume %s(%s) from the node %s: %v", name, req.VolumeId, req.NodeId, err)
	}

	log.Printf("volume %s(%s) has been detached from the node %s", name, req.VolumeId, req.NodeId)

	resp := &csi.ControllerUnpublishVolumeResponse{}

	log.Printf("ControllerUnpublishVolume response: %v", resp)
	return resp, nil
}

// GetCapacity returns the capacity of the storage
func (drv *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	log.Printf("GetCapacity request: %v", req)

	capacity, err := drv.getCapacity()
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "error getting capacity: %v", err)
	}

	resp := &csi.GetCapacityResponse{AvailableCapacity: capacity}

	log.Printf("GetCapacity response: %v", resp)
	return resp, nil
}

// ListSnapshots returns a list of requested volume snapshots
func (drv *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListSnapshots is not implemented")
}

// CreateSnapshot creates new volume snapshot
func (drv *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateSnapshot is not implemented")
}

// DeleteSnapshot deletes volume snapshot
func (drv *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "DeleteSnapshot is not implemented")
}
