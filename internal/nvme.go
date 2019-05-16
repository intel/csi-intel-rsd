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
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	devMaxDelay = 10
)

type DeviceList struct {
	Devices []struct {
		DevicePath string `json:"DevicePath"`
	} `json:"Devices"`
}

type ControllerInfo struct {
	Subnqn string `json:"subnqn"`
}

// NVMe interface declares NVMe operations required by the RSD CSI driver
type NVMe interface {
	// Connect to NVMe subsystem
	Connect(transport, traddr, traddrfamily, trsvcid, nqn, hostnqn string) (string, error)
	// Disconnect from NVMe subystem
	Disconnect(device string) error
}

type nvme struct{}

func nvmeCommand(options []string) ([]byte, error) {
	out, err := exec.Command("nvme", options...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command failed: %v, command: 'nvme %s', output: %q",
			err, strings.Join(options, " "), string(out))
	}
	return out, nil
}

// findNVMeDevice uses 'nvme list' and 'id-ctrl' to find device by NQN
func findNVMeDevice(nqn string) (string, error) {
	// wait for device node to appear
	for delay := 1; delay < devMaxDelay; delay++ {

		out, err := nvmeCommand([]string{"list", "-o", "json"})
		if err != nil {
			return "", err
		}

		var deviceList DeviceList
		err = json.Unmarshal(out, &deviceList)
		if err != nil {
			return "", fmt.Errorf("Can't unmarshal 'nvme list -o json' output: %v", err)
		}

		for _, device := range deviceList.Devices {
			out, err = nvmeCommand([]string{"id-ctrl", device.DevicePath, "-o", "json"})
			if err != nil {
				return "", err
			}

			var controllerInfo ControllerInfo
			err = json.Unmarshal(out, &controllerInfo)
			if err != nil {
				return "", fmt.Errorf("Can't decode 'nvme id-ctrl %s -o json' output: %v", device.DevicePath, err)
			}

			if strings.TrimSpace(controllerInfo.Subnqn) == strings.TrimSpace(nqn) {
				return device.DevicePath, nil
			}
		}
		time.Sleep(time.Duration(delay) * time.Second)
	}

	return "", fmt.Errorf("can't find NVMe device by NQN %s", nqn)
}

func (n *nvme) Connect(transport, traddr, traddrfamily, trsvcid, nqn, hostnqn string) (string, error) {
	// nvme connect --transport rdma --traddr 192.168.1.1 --trsvcid 4420
	//              --nqn nqn.2014-08.org.nvmexpress:uuid:157f29ff-18d2-4784-872e-cbf51bf4701a
	//              --hostnqn nqn.2014-08.org.nvmexpress:uuid:265524c1-de5f-4b42-93df-e2b99fe02eb4
	//
	// --transport: network fabric being used for a NVMe-over-Fabrics network
	// --traddr: network address of the Controller
	// --trsvcid: the transport service id. For transports using IP addressing (e.g. rdma) this field is the port number
	// --nqn: NQN of the NVMe subsystem to connect to (volume entry point NQN in this case)
	// --hostnqn: NQN of the host (computer system NQN in this case)
	options := []string{
		"connect",
		"--transport", transport,
		"--traddr", traddr,
		"--trsvcid", trsvcid,
		"--nqn", nqn,
		"--hostnqn", hostnqn,
	}
	if _, err := nvmeCommand(options); err != nil {
		return "", err
	}

	return findNVMeDevice(nqn)
}

func (n *nvme) Disconnect(device string) error {
	// nvme disconnect --device /dev/nvme1n1
	// --device: NVMe device
	_, err := nvmeCommand([]string{"disconnect", "--device", device})
	return err
}
