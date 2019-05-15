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
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	csirsd "github.com/intel/csi-intel-rsd/internal"
	"github.com/intel/csi-intel-rsd/pkg/rsd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	rsdUsernameEnv string = "rsd-username"
	rsdPasswordEnv string = "rsd-password"
	kubeNodeEnv    string = "KUBE_NODE_NAME"
	rsdNodeLabel   string = "csi.intel.com/rsd-node"
)

// Get current Kubernetes node label by name
func getLabel(name string) (string, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	nodeName := os.Getenv(kubeNodeEnv)
	if nodeName == "" {
		return "", fmt.Errorf("environment variable %s is not set", kubeNodeEnv)
	}

	node, err := clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("can't get node %s: %v", nodeName, err)
	}

	label, exists := node.GetLabels()[name]
	if !exists {
		return "", fmt.Errorf("Label %s is not set for a node %s", name, nodeName)
	}

	return label, nil
}

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

	var err error
	if *nodeID == "" {
		*nodeID, err = getLabel(rsdNodeLabel)
		if err != nil {
			log.Fatalf("Can't get RSD node ID: %v", err)
		}
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
