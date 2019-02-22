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
)

// Volume contains mapping between CSI and RSD volumes and internal driver information about a volume status
type Volume struct {
	CSIVolume       *csi.Volume
	RSDVolume       *rsd.Volume
	Name            string
	NodeID          string
	ISStaged        bool
	ISPublished     bool
	StageTargetPath string
	TargetPath      string
}

// Driver implements the following CSI interfaces:
//
//   csi.IdentityServer
//   csi.ControllerServer
//   csi.NodeServer
//
type Driver struct {
	sync.Mutex
	endpoint string
	srv      *grpc.Server

	rsdClient rsd.Transport

	volumes    map[string]*Volume
	volumesRWL sync.RWMutex

	// ready defines whether the driver is ready to function. This value will
	// be used by the `Identity` service via the `Probe()` method.
	ready   bool
	readyMu sync.Mutex // protects ready
}

// NewDriver returns a CSI plugin that contains the necessary gRPC
// interfaces to interact with Kubernetes over unix domain socket
func NewDriver(ep string, rsdClient rsd.Transport) *Driver {
	return &Driver{
		endpoint:  ep,
		rsdClient: rsdClient,
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
			log.Fatalf("method %s failed, error: %s", info.FullMethod, err)
		}
		return resp, err
	}

	drv.srv = grpc.NewServer(grpc.UnaryInterceptor(errHandler))
	csi.RegisterIdentityServer(drv.srv, drv)
	csi.RegisterControllerServer(drv.srv, drv)
	//csi.RegisterNodeServer(drv.srv, drv)

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

	strID := strconv.Itoa(rsdVolume.ID)

	capacityBytes, err := strconv.ParseInt(rsdVolume.CapacityBytes, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("can't convert CapacityBytes %s to int: %v", rsdVolume.CapacityBytes, err)
	}

	csiVolume := &csi.Volume{
		VolumeId:      strID,
		VolumeContext: map[string]string{"name": name},
		CapacityBytes: capacityBytes,
	}

	drv.volumes[name] = &Volume{
		CSIVolume:       csiVolume,
		RSDVolume:       rsdVolume,
		NodeID:          "",
		ISStaged:        false,
		ISPublished:     false,
		StageTargetPath: "",
		TargetPath:      "",
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

// DeleteVolume deletes RSD volume using RSD API
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
			return fmt.Errorf("can't delete RSD Volume %d: %v", vol.RSDVolume.ID, err)
		}

		// delete volume from the map
		delete(drv.volumes, name)
	}
	return nil
}
