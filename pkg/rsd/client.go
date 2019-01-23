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

package rsd

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Transport is an interface to communicate with RSD server
type Transport interface {
	Get(entrypoint string, result interface{}) error
}

// Client is a struct that interfaces with the RSD Redfish API
type Client struct {
	baseurl    string
	username   string
	password   string
	httpClient *http.Client
}

// NewClient creates new RSD Client
func NewClient(baseurl, username, password string, timeout time.Duration) (*Client, error) {
	return &Client{
		baseurl:    baseurl,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// Get queries RSD endpoing and returns decoded http response
func (rsd *Client) Get(entrypoint string, result interface{}) error {
	url := rsd.baseurl + entrypoint
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errors.Wrapf(err, "Can't make request from %s", url)
	}

	if rsd.username != "" {
		req.SetBasicAuth(rsd.username, rsd.password)
	}

	resp, err := rsd.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "Can't get http response from %s", url)
	}

	defer resp.Body.Close()

	// Decode response
	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return errors.Wrapf(err, "Can't decode http response from %s", url)
	}

	return nil
}

// GetStorageServiceCollection returns StorageServiceCollection
func GetStorageServiceCollection(rsd Transport) (*StorageServiceCollection, error) {
	var result StorageServiceCollection
	err := rsd.Get(StorageServiceCollectionEntryPoint, &result)
	if err != nil {
		return nil, errors.Wrap(err, "Can't query StorageServiceCollection")
	}

	return &result, err
}
