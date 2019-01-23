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

package main

import (
	"flag"
	"log"
	"time"

	"github.com/intel/csi-intel-rsd/internal"
	"github.com/intel/csi-intel-rsd/pkg/rsd"
)

func main() {
	// Parse command line
	endpoint := flag.String("endpoint", "unix:///var/lib/kubelet/plugins/csi-intel-rsd.sock", "CSI endpoint")
	username := flag.String("username", "", "User name")
	password := flag.String("password", "", "Password")
	baseurl := flag.String("baseurl", "http://localhost:2443", "Redfish URL")
	timeout := flag.Duration("timeout", 10*time.Second, "HTTP timeout")
	flag.Parse()

	rsdClient, err := rsd.NewClient(*baseurl, *username, *password, *timeout)
	if err != nil {
		log.Fatalln(err)
	}

	driver := csirsd.NewDriver(*endpoint, rsdClient)

	if err := driver.Run(); err != nil {
		log.Fatalln(err)
	}
}
