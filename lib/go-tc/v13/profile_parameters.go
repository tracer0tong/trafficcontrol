package v13

import tc "github.com/apache/trafficcontrol/lib/go-tc"

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

// ProfileParametersResponse ...
type ProfileParametersResponse struct {
	Response []ProfileParameter `json:"response"`
}

// A Single ProfileParameter Response for Create to depict what changed
// swagger:response ProfileParameterResponse
// in: body
type ProfileParameterResponse struct {
	// in: body
	Response ProfileParameter `json:"response"`
}

// ProfileParameter ...
type ProfileParameter struct {
	LastUpdated tc.TimeNoMod `json:"lastUpdated"`
	Profile     string       `json:"profile"`
	ProfileID   int          `json:"profileId"`
	Parameter   string       `json:"parameter"`
	ParameterID int          `json:"parameterId"`
}

// ProfileParameterNullable ...
type ProfileParameterNullable struct {
	LastUpdated *tc.TimeNoMod `json:"lastUpdated" db:"last_updated"`
	Profile     *string       `json:"profile" db:"profile"`
	ProfileID   *int          `json:"profileId" db:"profile_id"`
	Parameter   *string       `json:"parameter" db:"parameter"`
	ParameterID *int          `json:"parameterId" db:"parameter_id"`
}
