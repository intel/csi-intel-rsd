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
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/intel/csi-intel-rsd/pkg/rsd"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	// DriverName defines the name that is used in Kubernetes and the CSI
	// system for the canonical, official name of this plugin
	DriverName = "csi.rsd.intel.com"

	// DriverVersion defines current CSI Driver version
	DriverVersion = "0.0.1"

	// PublishInfoVolumeName is used to pass the volume name from
	// `ControllerPublishVolume` to `NodeStageVolume or `NodePublishVolume`
	PublishInfoVolumeName = DriverName + "/volume-name"
)

type endPointInfo struct {
	ipAddress         string
	ipAddressFamily   string
	ipPort            int
	transportProtocol string
	nqn               string
}

// Volume contains mapping between CSI and RSD volumes and internal driver information about a volume status
type Volume struct {
	Name        string
	CSIVolume   *csi.Volume
	RSDVolume   *rsd.Volume
	EndPoint    *endPointInfo
	RSDNodeID   string
	RSDNodeNQN  string
	Device      string
	IsPublished bool
	IsStaged    bool
}

// Driver implements the following CSI interfaces:
//
//   csi.IdentityServer
//   csi.ControllerServer
//   csi.NodeServer
//
type Driver struct {
	sync.Mutex
	endpoint  string
	srv       *grpc.Server
	RSDNodeID string

	rsdClient rsd.Transport
	mounter   Mounter
	nvme      NVMe

	volumes    map[string]*Volume
	volumesRWL sync.RWMutex

	// ready defines whether the driver is ready to function. This value will
	// be used by the `Identity` service via the `Probe()` method.
	ready   bool
	readyMu sync.Mutex // protects ready
}

// NewDriver returns a CSI plugin that contains the necessary gRPC
// interfaces to interact with Kubernetes over unix domain socket
func NewDriver(ep string, RSDNodeID string, rsdClient rsd.Transport) *Driver {
	return &Driver{
		endpoint:  ep,
		RSDNodeID: RSDNodeID,
		rsdClient: rsdClient,
		mounter:   &mounter{},
		nvme:      &nvme{},
		volumes:   map[string]*Volume{},
	}
}

// Run starts the CSI plugin by communication over the given endpoint
func (drv *Driver) Run() error {
	u, err := url.Parse(drv.endpoint)
	if err != nil {
		return fmt.Errorf("unable to parse address: %v", err)
	}

	spath := path.Join(u.Host, filepath.FromSlash(u.Path))
	if u.Host == "" {
		spath = filepath.FromSlash(u.Path)
	}

	// CSI plugins talk only over UNIX sockets currently
	if u.Scheme != "unix" {
		return fmt.Errorf("currently only unix domain sockets are supported, have: %s", u.Scheme)
	}

	// remove the socket if it's already there. This can happen if we
	// deploy a new version and the socket was created from the old running
	// plugin.
	if _, err = os.Stat(spath); !os.IsNotExist(err) {
		log.Printf("removing socket %s", spath)
		if err = os.Remove(spath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove unix domain socket file %s, error: %v", spath, err)
		}
	}

	listener, err := net.Listen(u.Scheme, spath)
	if err != nil {
		return fmt.Errorf("failed to listen socket %s: %v", spath, err)
	}

	// log response errors
	errHandler := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			log.Printf("method %s failed, error: %s", info.FullMethod, err)
		}
		return resp, err
	}

	drv.srv = grpc.NewServer(grpc.UnaryInterceptor(errHandler))
	csi.RegisterIdentityServer(drv.srv, drv)
	csi.RegisterControllerServer(drv.srv, drv)
	csi.RegisterNodeServer(drv.srv, drv)

	drv.ready = true
	log.Printf("server started serving on %s", drv.endpoint)
	return drv.srv.Serve(listener)
}

// List existing volumes sorted by name
func (drv *Driver) listCSIVolumes() []*csi.Volume {
	// sort volume names
	keys := make([]string, 0, len(drv.volumes))
	for k := range drv.volumes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// collect CSI volumes ordered by name
	csiVolumes := []*csi.Volume{}
	for _, k := range keys {
		csiVolumes = append(csiVolumes, drv.volumes[k].CSIVolume)
	}
	return csiVolumes
}

// Creates new volume and adds it to the Volumes map
func (drv *Driver) newVolume(name string, requiredCapacity int64) (*csi.Volume, error) {
	if _, exists := drv.volumes[name]; exists {
		return nil, fmt.Errorf("failed attempt to create exisiting volume %s", name)
	}

	// Volume doesn't exist - create new one

	// Get volume collection
	client := drv.rsdClient
	volCollection, err := rsd.GetVolumeCollection(client, 0)
	if err != nil {
		return nil, err
	}

	// Create new RSD volume
	rsdVolume, err := volCollection.NewVolume(client, requiredCapacity)
	if err != nil {
		return nil, err
	}

	csiVolume := &csi.Volume{
		VolumeId:      rsdVolume.ID,
		VolumeContext: map[string]string{"name": name},
		CapacityBytes: rsdVolume.CapacityBytes,
	}

	drv.volumes[name] = &Volume{
		Name:      name,
		CSIVolume: csiVolume,
		RSDVolume: rsdVolume,
		RSDNodeID: "",
	}

	return csiVolume, nil
}

func (drv *Driver) findCSIVolumeByName(name string) *csi.Volume {
	if vol, exists := drv.volumes[name]; exists {
		return vol.CSIVolume
	}
	return nil
}

func (drv *Driver) findVolByID(volumeID string) (string, *Volume) {
	for name, vol := range drv.volumes {
		if vol.CSIVolume.VolumeId == volumeID {
			return name, vol
		}
	}
	return "", nil
}

// deleteVolume deletes RSD volume using RSD API
// and removes volume from the internal map drv.volumes
// It does nothing if volume doesn't exist
func (drv *Driver) deleteVolume(volumeID string) error {
	drv.volumesRWL.Lock()
	defer drv.volumesRWL.Unlock()

	name, vol := drv.findVolByID(volumeID)
	if name != "" {
		// delete RSD volume
		err := vol.RSDVolume.Delete(drv.rsdClient)
		if err != nil {
			return fmt.Errorf("can't delete RSD Volume %s: %v", vol.RSDVolume.ID, err)
		}

		// delete volume from the map
		delete(drv.volumes, name)
	}
	return nil
}

func findTransportDetails(endPoint *rsd.EndPoint) *endPointInfo {
	for _, ipTransportDetail := range endPoint.IPTransportDetails {
		proto := strings.ToUpper(ipTransportDetail.TransportProtocol)
		if proto != "ROCE" && proto != "ROCEV2" {
			continue
		}
		if ipTransportDetail.IPv4Address.Address != "" {
			return &endPointInfo{
				ipAddress:         ipTransportDetail.IPv4Address.Address,
				ipAddressFamily:   "IPv4",
				ipPort:            ipTransportDetail.Port,
				transportProtocol: "rdma",
			}
		}
		if ipTransportDetail.IPv6Address.Address != "" {
			return &endPointInfo{
				ipAddress:         ipTransportDetail.IPv6Address.Address,
				ipAddressFamily:   "IPv6",
				ipPort:            ipTransportDetail.Port,
				transportProtocol: "rdma",
			}
		}
	}
	return nil
}

func findEndPointInfo(endPoints []*rsd.EndPoint) *endPointInfo {
	for _, endPoint := range endPoints {
		epi := findTransportDetails(endPoint)
		if epi != nil {
			epi.nqn = endPoint.GetNQN()
			return epi
		}
	}
	return nil
}

// getVolumeEndPointInfo gets RSD EndPoint and validates it
func (drv *Driver) getVolumeEndPointInfo(volume *Volume) (*endPointInfo, error) {
	// Get Entry Point associated with this RSD volume
	endPoints, err := volume.RSDVolume.GetEndPoints(drv.rsdClient)
	if err != nil {
		return nil, err
	}
	if len(endPoints) == 0 {
		return nil, fmt.Errorf("no RSD Endpoints found for the volume %s", volume.Name)
	}

	epi := findEndPointInfo(endPoints)
	if epi == nil {
		return nil, fmt.Errorf("no suitable RSD endpoints found for the volume %s", volume.Name)
	}
	return epi, nil
}

// getComputerSystemNQN gets NQN of the Computer System
func (drv *Driver) getComputerSystemNQN(computerSystem *rsd.ComputerSystem) (string, error) {
	endPoints, err := computerSystem.GetEndPoints(drv.rsdClient)
	if err != nil {
		return "", err
	}

	if len(endPoints) == 0 {
		return "", fmt.Errorf("no RSD Endpoints found for the computer system %s", computerSystem.Name)
	}

	for _, endPoint := range endPoints {
		nqn := endPoint.GetNQN()
		if nqn != "" {
			return nqn, nil
		}
	}

	return "", fmt.Errorf("no NQN found for the computer system %s", computerSystem.Name)
}

// publishVolume publishes volume on the node
func (drv *Driver) publishVolume(volume *Volume, RSDNodeID string) error {
	if volume.IsPublished {
		return nil
	}
	node, err := rsd.GetNode(drv.rsdClient, RSDNodeID)
	if err != nil {
		return err
	}

	// Attach RSD volume to the node
	err = node.AttachResource(drv.rsdClient, volume.RSDVolume.OdataID)
	if err != nil {
		return err
	}

	// Read volume info again as volume endpoint appears only after attachment
	volume.RSDVolume, err = rsd.GetVolume(drv.rsdClient, 0, volume.RSDVolume.ID)
	if err != nil {
		return err
	}

	// Get endpoint associated with this RSD volume
	volume.EndPoint, err = drv.getVolumeEndPointInfo(volume)
	if err != nil {
		return err
	}

	// Get Computer System associated with the node
	var computerSystem rsd.ComputerSystem
	err = rsd.GetByOdataID(drv.rsdClient, node.Links.ComputerSystem.OdataID, &computerSystem)
	if err != nil {
		return err
	}

	// Get NQN of this Computer System
	volume.RSDNodeNQN, err = drv.getComputerSystemNQN(&computerSystem)
	if err != nil {
		return err
	}

	volume.RSDNodeID = RSDNodeID
	volume.IsPublished = true

	return nil
}

// unpublishVolume unpublishes volume from the node
func (drv *Driver) unpublishVolume(volume *Volume, RSDNodeID string) error {
	if !volume.IsPublished {
		return nil
	}
	node, err := rsd.GetNode(drv.rsdClient, RSDNodeID)
	if err != nil {
		return err
	}

	// Detach RSD volume to the node
	err = node.DetachResource(drv.rsdClient, volume.RSDVolume.OdataID)
	if err != nil {
		return err
	}

	volume.RSDNodeNQN = ""
	volume.RSDNodeID = ""
	volume.IsPublished = false

	return nil
}

// nodeStageVolume connects the volume to the node using nvme connect and mounts it to the Target Staging path
func (drv *Driver) nodeStageVolume(volume *Volume, fsType, stagingTargetPath string, mountOpts []string) error {
	if !volume.IsPublished {
		return fmt.Errorf("nodeStageVolume: volume %s is not published", volume.Name)
	}

	if volume.IsStaged {
		return nil
	}

	ep := volume.EndPoint
	if ep == nil {
		return fmt.Errorf("nodeStageVolume: no endpoint found for volume %s", volume.Name)
	}

	dev, err := drv.nvme.Connect(
		ep.transportProtocol,
		ep.ipAddress,
		ep.ipAddressFamily,
		strconv.Itoa(ep.ipPort),
		ep.nqn,
		volume.RSDNodeNQN)

	if err != nil {
		return err
	}

	formatted, err := drv.mounter.IsFormatted(dev)
	if err != nil {
		return err
	}

	if !formatted {
		if err := drv.mounter.Format(dev, fsType); err != nil {
			return err
		}
	}

	mounted, err := drv.mounter.IsMounted(dev, stagingTargetPath)
	if err != nil {
		return err
	}

	if !mounted {
		err = drv.mounter.Mount(dev, stagingTargetPath, fsType, mountOpts...)
		if err != nil {
			return err
		}
	}

	volume.Device = dev
	volume.IsStaged = true
	return nil
}

// nodeUnstageVolume unmounts the volume from the Staging Target path
func (drv *Driver) nodeUnstageVolume(volume *Volume, stagingTargetPath string) error {
	if !volume.IsStaged {
		return nil
	}
	mounted, err := drv.mounter.IsMounted("", stagingTargetPath)
	if err != nil {
		return err
	}

	if mounted {
		err := drv.mounter.Unmount(stagingTargetPath)
		if err != nil {
			return err
		}
	}

	if err = drv.nvme.Disconnect(volume.Device); err != nil {
		return err
	}

	volume.Device = ""
	volume.IsStaged = false
	return nil
}

// nodePublishVolume bind-mounts Staging directory to the Target Path
func (drv *Driver) nodePublishVolume(volume *Volume, fsType, stagingTargetPath, targetPath string, mountOpts []string) error {
	mounted, err := drv.mounter.IsMounted(stagingTargetPath, targetPath)
	if err != nil {
		return err
	}

	if !mounted {
		if err := drv.mounter.Mount(stagingTargetPath, targetPath, fsType, mountOpts...); err != nil {
			return err
		}
	}

	return nil
}

// nodeUnpublishVolume unmounts the volume from the Target Path
func (drv *Driver) nodeUnpublishVolume(volume *Volume, targetPath string) error {
	mounted, err := drv.mounter.IsMounted("", targetPath)
	if err != nil {
		return err
	}

	if mounted {
		err := drv.mounter.Unmount(targetPath)
		if err != nil {
			return err
		}
	}

	return err
}
