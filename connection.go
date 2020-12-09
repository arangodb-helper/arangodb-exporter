//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Adam Janikowski
//

package main

import (
	"io/ioutil"
	"strings"
)

type Authentication func() (string, error)

func newAuthentication() Authentication {
	if arangodbOptions.jwtFile != "" {
		return func() (string, error) {

			data, err := ioutil.ReadFile(arangodbOptions.jwtFile)
			if err != nil {
				return "", err
			}

			return strings.TrimSpace(string(data)), nil
		}
	} else if arangodbOptions.jwtSecret != "" {
		return func() (string, error) {
			return CreateArangodJWT(arangodbOptions.jwtSecret)
		}
	}
	return func() (string, error) {
		return "", nil
	}
}
