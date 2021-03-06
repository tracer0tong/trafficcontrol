package crconfig

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

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

// Handler creates and serves the CRConfig from the raw SQL data.
// This MUST only be used for debugging or previewing, the raw un-snapshotted data MUST NOT be used by any component of the CDN.
func Handler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"cdn"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	start := time.Now()
	crConfig, err := Make(inf.Tx.Tx, inf.Params["cdn"], inf.User.UserName, r.Host, r.URL.Path, inf.Config.Version)
	if err != nil {
		api.HandleErr(w, r, http.StatusInternalServerError, nil, err)
		return
	}
	log.Infof("CRConfig time to generate: %+v\n", time.Since(start))
	*inf.CommitTx = true
	api.WriteResp(w, r, crConfig)
}

// SnapshotGetHandler gets and serves the CRConfig from the snapshot table.
func SnapshotGetHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"cdn"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	snapshot, cdnExists, err := GetSnapshot(inf.Tx.Tx, inf.Params["cdn"])
	if err != nil {
		api.HandleErr(w, r, http.StatusInternalServerError, nil, errors.New("getting snapshot: "+err.Error()))
		return
	}
	if !cdnExists {
		api.HandleErr(w, r, http.StatusNotFound, errors.New("CDN not found"), nil)
		return
	}
	*inf.CommitTx = true
	w.Header().Set(tc.ContentType, tc.ApplicationJson)
	w.Write([]byte(`{"response":` + snapshot + `}`))
}

// SnapshotOldGetHandler gets and serves the CRConfig from the snapshot table, not wrapped in response to match the old non-API CRConfig-Snapshots endpoint
func SnapshotOldGetHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"cdn"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	snapshot, cdnExists, err := GetSnapshot(inf.Tx.Tx, inf.Params["cdn"])
	if err != nil {
		api.HandleErr(w, r, http.StatusInternalServerError, nil, errors.New("getting snapshot: "+err.Error()))
		return
	}
	if !cdnExists {
		api.HandleErr(w, r, http.StatusNotFound, errors.New("CDN not found"), nil)
		return
	}
	*inf.CommitTx = true
	w.Header().Set(tc.ContentType, tc.ApplicationJson)
	w.Write([]byte(snapshot))
}

// SnapshotHandler creates the CRConfig JSON and writes it to the snapshot table in the database.
func SnapshotHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	cdn, ok := inf.Params["cdn"]
	if !ok {
		id, ok := inf.IntParams["id"]
		if !ok {
			api.HandleErr(w, r, http.StatusNotFound, errors.New("params missing CDN"), nil)
			return
		}
		name, ok, err := getCDNNameFromID(id, inf.Tx.Tx)
		if err != nil {
			api.HandleErr(w, r, http.StatusInternalServerError, nil, errors.New("Error getting CDN name from ID: "+err.Error()))
			return
		}
		if !ok {
			api.HandleErr(w, r, http.StatusNotFound, errors.New("No CDN found with that ID"), nil)
			return
		}
		cdn = name
	}

	crConfig, err := Make(inf.Tx.Tx, cdn, inf.User.UserName, r.Host, r.URL.Path, inf.Config.Version)
	if err != nil {
		api.HandleErr(w, r, http.StatusInternalServerError, nil, err)
		return
	}

	if err := Snapshot(inf.Tx.Tx, crConfig); err != nil {
		api.HandleErr(w, r, http.StatusInternalServerError, nil, errors.New(r.RemoteAddr+" snaphsotting CRConfig: "+err.Error()))
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "Snapshot of CRConfig performed for "+cdn, inf.User, inf.Tx.Tx)
	*inf.CommitTx = true
	w.WriteHeader(http.StatusOK) // TODO change to 204 No Content in new version
}

// SnapshotGUIHandler creates the CRConfig JSON and writes it to the snapshot table in the database. The response emulates the old Perl UI function. This should go away when the old Perl UI ceases to exist.
func SnapshotOldGUIHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, _ := api.NewInfo(r, []string{"cdn"}, nil)
	if userErr != nil || sysErr != nil {
		log.Errorln(r.RemoteAddr + " unable to get info from request: " + sysErr.Error())
		writePerlHTMLErr(w, r, userErr)
		return
	}
	defer inf.Close()

	crConfig, err := Make(inf.Tx.Tx, inf.Params["cdn"], inf.User.UserName, r.Host, r.URL.Path, inf.Config.Version)
	if err != nil {
		log.Errorln(r.RemoteAddr + " making CRConfig: " + err.Error())
		writePerlHTMLErr(w, r, err)
		return
	}

	if err := Snapshot(inf.Tx.Tx, crConfig); err != nil {
		log.Errorln(r.RemoteAddr + " making CRConfig: " + err.Error())
		writePerlHTMLErr(w, r, err)
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "Snapshot of CRConfig performed for "+inf.Params["cdn"], inf.User, inf.Tx.Tx)
	*inf.CommitTx = true
	http.Redirect(w, r, "/tools/flash_and_close/"+url.PathEscape("Successfully wrote the CRConfig.json!"), http.StatusFound)
}

func writePerlHTMLErr(w http.ResponseWriter, r *http.Request, err error) {
	http.Redirect(w, r, "/tools/flash_and_close/"+url.PathEscape("Error: "+err.Error()), http.StatusFound)
}
