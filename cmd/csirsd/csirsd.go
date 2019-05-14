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
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	csirsd "github.com/intel/csi-intel-rsd/internal"
	"github.com/intel/csi-intel-rsd/pkg/rsd"
)

const (
	rsdUsernameEnv string = "rsd-username"
	rsdPasswordEnv string = "rsd-password"
)

func main() {
	// Parse command line
	endpoint := flag.String("endpoint", "unix:///var/lib/kubelet/plugins/csi-intel-rsd.sock", "CSI endpoint")
	username := flag.String("username", os.Getenv(rsdUsernameEnv), "RSD username")
	password := flag.String("password", os.Getenv(rsdPasswordEnv), "RSD password")
	baseurl := flag.String("baseurl", "http://localhost:2443", "Redfish URL")
	nodeID := flag.String("nodeid", "", "RSD Node id")
	timeout := flag.Duration("timeout", 10*time.Second, "HTTP timeout")
	insecure := flag.Bool("insecure", false, "allow connections to https RSD without certificate verification")
	flag.Parse()

	// uset RSD access creds for security reasons
	os.Unsetenv(rsdUsernameEnv)
	os.Unsetenv(rsdPasswordEnv)

	if *nodeID == "" {
		log.Fatal("nodeid mush be provided")
	}

	httpClient := &http.Client{Timeout: *timeout}
	if *insecure {
		httpClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}

	rsdClient, err := rsd.NewClient(*baseurl, *username, *password, httpClient)
	if err != nil {
		log.Fatalln(err)
	}

	driver := csirsd.NewDriver(*endpoint, *nodeID, rsdClient)

	if err := driver.Run(); err != nil {
		log.Fatalln(err)
	}
}
