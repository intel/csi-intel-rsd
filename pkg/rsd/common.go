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

import "github.com/pkg/errors"

type endPointOdataID struct {
	OdataID string `json:"@odata.id"`
}

// GetByOdataID gets resource by its ODataID
func GetByOdataID(rsd Transport, oDataID string, result interface{}) error {
	err := rsd.Get(oDataID, result)
	if err != nil {
		return errors.Wrapf(err, "can't query resource by ODataID %s", oDataID)
	}

	return nil
}

// GetEndPoints returns List of EndPoints associated with a Volume
func GetEndPoints(rsd Transport, endPointOdataIDs []endPointOdataID) ([]*EndPoint, error) {
	var result []*EndPoint
	for _, ep := range endPointOdataIDs {
		epURL := ep.OdataID
		endPoint := EndPoint{}
		err := rsd.Get(epURL, &endPoint)
		if err != nil {
			return nil, errors.Wrapf(err, "can't query EndPoint %s", epURL)
		}
		result = append(result, &endPoint)
	}
	return result, nil
}
