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
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	sysfsDirectory = "/sys/class/nvme"
)

// NVMe interface declares NVMe operations required by the RSD CSI driver
type NVMe interface {
	// Connect to NVMe subsystem
	Connect(transport, traddr, traddrfamily, trsvcid, nqn string) (string, error)
	// Disconnect from NVMe subystem
	Disconnect(device string) error
}

type nvme struct{}

func nvmeCommand(options []string) error {
	out, err := exec.Command("nvme", options...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %v, command: 'nvme %s', output: %q",
			err, strings.Join(options, " "), string(out))
	}
	return nil
}

// findNVMeDevice scans /sys/class/nvme/ to find device by NQN
func findNVMeDevice(nqn string) (string, error) {
	entries, err := ioutil.ReadDir(sysfsDirectory)
	if err != nil {
		return "", fmt.Errorf("can't read sysfs direcroty %s. Kernel driver not loaded?", sysfsDirectory)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subsysnqnPath := path.Join(sysfsDirectory, entry.Name(), "subsysnqn ")
		if _, err := os.Stat(subsysnqnPath); !os.IsNotExist(err) {
			content, err := ioutil.ReadFile(subsysnqnPath)
			if err != nil {
				return "", fmt.Errorf("can't read %s: %v", subsysnqnPath, err)
			}
			if strings.TrimSpace(string(content)) == strings.TrimSpace(nqn) {
				// found volume nqn in the /sys/class/nvme/nvmeX/subsysnqn
				// the device name should be a subdirectory started with nvmeX
				nvmeDir := path.Join(sysfsDirectory, entry.Name())
				nvmeEntries, err := ioutil.ReadDir(nvmeDir)
				if err != nil {
					return "", fmt.Errorf("can't read sysfs nvme directory %s", nvmeDir)
				}
				for _, nvmeEntry := range nvmeEntries {
					if nvmeEntry.IsDir() && strings.HasPrefix(nvmeEntry.Name(), entry.Name()) {
						return nvmeEntry.Name(), nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("can't found NVMe device in %s by NQN %s", sysfsDirectory, nqn)
}

func (n *nvme) Connect(transport, traddr, traddrfamily, trsvcid, nqn string) (string, error) {
	// nvme connect --transport rdma --nqn nqn.2014-08.org.nvmexpress:uuid:157f29ff-18d2-4784-872e-cbf51bf4701a
	//              --traddr 192.168.121.167 --trsvcid 4420
	// --transport: network fabric being used for a NVMe-over-Fabrics network
	// --traddr: network address of the Controller
	// --trsvcid: the transport service id. For transports using IP addressing (e.g. rdma) this field is the port number
	// --nqn: name for the NVMe subsystem to connect to
	options := []string{"connect", "--transport", transport, "--traddr", traddr, "--trsvcid", trsvcid, "--nqn", nqn}
	if err := nvmeCommand(options); err != nil {
		return "", err
	}

	return findNVMeDevice(nqn)
}

func (n *nvme) Disconnect(device string) error {
	// nvme disconnect --device /dev/nvme1n1
	// --device: NVMe device
	return nvmeCommand([]string{"disconnect", "--device", device})
}
