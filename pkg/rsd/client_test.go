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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetStorageServiceCollection(t *testing.T) {
	var tcases = []struct {
		name    string
		isError bool
		fname   string
	}{
		{
			name:    "Success",
			isError: false,
			fname:   "testdata/storageservices-valid.json",
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if req.URL.String() != StorageServiceCollectionEntryPoint {
					t.Errorf("Unexpected URL: %s, should be: %s", req.URL.String(), StorageServiceCollectionEntryPoint)
				}
				content, err := ioutil.ReadFile(tc.fname)
				if err != nil {
					t.Fatalf("can't read file %s: %v", tc.fname, err)
				}
				rw.Write(content)
			}))
			defer server.Close()

			rsdClient, err := NewClient(server.URL, "", "", 10*time.Second)
			if err != nil {
				t.Fatalf("%+v", err)
			}

			ssCollection, err := GetStorageServiceCollection(rsdClient)
			if err == nil && tc.isError {
				t.Error("unexpected success")
			}
			nMembers := len(ssCollection.Members)
			if nMembers != 1 {
				t.Errorf("unexpected amount of members: %d, should be 1", nMembers)
			}
		})
	}
}
