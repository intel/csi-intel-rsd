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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Mounter interface declares volume mounting and formatting operations
type Mounter interface {
	// Mount mounts source to target as fstype with given options.
	Mount(source string, target string, fstype string, opts ...string) error
	// Unmount unmounts given target.
	Unmount(target string) error
	/// IsMounted checks whether the source device is mounted to the target
	// path. Source can be empty. In that case it only checks whether the
	// device is mounted or not.
	// It returns true if it's mounted.
	IsMounted(source, target string) (bool, error)
	// IsFormatted checks whether the source device is formatted or not. It
	// returns true if the source device is already formatted.
	IsFormatted(source string) (bool, error)
	// Format formats the source with the given filesystem type
	Format(source, fsType string) error
}

type mounter struct{}

func (m *mounter) Mount(source, target, fsType string, opts ...string) error {
	if fsType == "" {
		return errors.New("fs type is not specified for mounting the volume")
	}

	if source == "" {
		return errors.New("source is not specified for mounting the volume")
	}

	if target == "" {
		return errors.New("target is not specified for mounting the volume")
	}

	mountArgs := []string{"-t", fsType}

	if len(opts) > 0 {
		mountArgs = append(mountArgs, "-o", strings.Join(opts, ","))
	}

	mountArgs = append(mountArgs, source)
	mountArgs = append(mountArgs, target)

	// create target, os.Mkdirall is noop if it exists
	err := os.MkdirAll(target, 0750)
	if err != nil {
		return err
	}

	out, err := exec.Command("mount", mountArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mounting failed: %v cmd: 'mount %s' output: %q", err, strings.Join(mountArgs, " "), string(out))
	}

	return nil
}

func (m *mounter) Unmount(target string) error {
	if target == "" {
		return errors.New("target is not specified for unmounting the volume")
	}

	out, err := exec.Command("umount", target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmounting failed: %v cmd: 'umount %s' output: %q",
			err, target, string(out))
	}

	return nil
}

func (m *mounter) IsFormatted(source string) (bool, error) {
	if source == "" {
		return false, errors.New("source is not specified")
	}

	lsblkCmd := "lsblk"
	_, err := exec.LookPath(lsblkCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return false, fmt.Errorf("%q executable not found in $PATH", lsblkCmd)
		}
		return false, err
	}

	lsblkArgs := []string{"-n", "-o", "FSTYPE", source}
	out, err := exec.Command(lsblkCmd, lsblkArgs...).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("checking formatting failed: %v cmd: %q %s, output: %q",
			err, lsblkCmd, strings.Join(lsblkArgs, " "), string(out))
	}

	if strings.TrimSpace(string(out)) == "" {
		return false, nil
	}

	return true, nil
}

func (m *mounter) IsMounted(source, target string) (bool, error) {
	findmntCmd := "findmnt"
	_, err := exec.LookPath(findmntCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return false, fmt.Errorf("%q executable not found in $PATH", findmntCmd)
		}
		return false, err
	}

	findmntArgs := []string{"--mountpoint", target}
	if source != "" {
		findmntArgs = append(findmntArgs, "--source", source)
	}

	out, err := exec.Command(findmntCmd, findmntArgs...).CombinedOutput()
	if err != nil {
		// findmnt exits with non zero exit status if it couldn't find anything
		if strings.TrimSpace(string(out)) == "" {
			return false, nil
		}

		return false, fmt.Errorf("checking mounted failed: %v cmd: %q output: %q",
			err, findmntCmd, string(out))
	}

	if strings.TrimSpace(string(out)) == "" {
		return false, nil
	}

	return true, nil
}

func (m *mounter) Format(source, fsType string) error {
	mkfsCmd := fmt.Sprintf("mkfs.%s", fsType)

	_, err := exec.LookPath(mkfsCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return fmt.Errorf("%q executable not found in $PATH", mkfsCmd)
		}
		return err
	}

	mkfsArgs := []string{}

	if fsType == "" {
		return errors.New("fs type is not specified for formatting the volume")
	}

	if source == "" {
		return errors.New("source is not specified for formatting the volume")
	}

	mkfsArgs = append(mkfsArgs, source)
	if fsType == "ext4" || fsType == "ext3" {
		mkfsArgs = []string{"-F", source}
	}

	out, err := exec.Command(mkfsCmd, mkfsArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("formatting disk failed: %v cmd: '%s %s' output: %q",
			err, mkfsCmd, strings.Join(mkfsArgs, " "), string(out))
	}

	return nil
}
