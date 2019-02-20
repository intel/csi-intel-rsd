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
	"reflect"
	"sync"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/ptypes/wrappers"
)

func TestDriver_GetPluginInfo(t *testing.T) {
	t.Run("Test GetPluginInfo", func(t *testing.T) {
		drv := &Driver{}
		got, err := drv.GetPluginInfo(context.Background(), &csi.GetPluginInfoRequest{})
		if err != nil {
			t.Errorf("Driver.GetPluginInfo() unexpected error = %v", err)
			return
		}
		expected := &csi.GetPluginInfoResponse{Name: DriverName, VendorVersion: DriverVersion}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("Driver.GetPluginInfo() = %v, want %v", got, expected)
		}
	})
}

func TestDriver_GetPluginCapabilities(t *testing.T) {
	t.Run("Test GetPluginCapabilities", func(t *testing.T) {
		drv := &Driver{}
		got, err := drv.GetPluginCapabilities(context.Background(), &csi.GetPluginCapabilitiesRequest{})
		if err != nil {
			t.Errorf("Driver.GetPluginCapabilities() unexpected error: %v", err)
			return
		}
		expected := &csi.GetPluginCapabilitiesResponse{
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
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("Driver.GetPluginInfo() = %v, want %v", got, expected)
		}
	})
}

func TestDriver_Probe(t *testing.T) {
	type fields struct {
		Mutex   sync.Mutex
		ready   bool
		readyMu sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
		want   *csi.ProbeResponse
	}{
		{name: "Ready", fields: fields{ready: true}, want: &csi.ProbeResponse{Ready: &wrappers.BoolValue{Value: true}}},
		{name: "Not ready", fields: fields{ready: false}, want: &csi.ProbeResponse{Ready: &wrappers.BoolValue{Value: false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drv := &Driver{
				Mutex:   tt.fields.Mutex,
				ready:   tt.fields.ready,
				readyMu: tt.fields.readyMu,
			}
			got, err := drv.Probe(context.Background(), &csi.ProbeRequest{})
			if err != nil {
				t.Errorf("Driver.Probe() unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Driver.Probe() = %v, want %+v", got, tt.want)
			}
		})
	}
}
