/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package nmisotp

import (
	"fmt"

	"mynewt.apache.org/newtmgr/nmxact/nmxutil"
	"mynewt.apache.org/newtmgr/nmxact/sesn"
)

type XportCfg struct {
	BusName        string
	RXAddr, TXAddr uint32
	Mtu            int
}

func NewXportCfg() *XportCfg {
	return &XportCfg{
		BusName: "can0",
		RXAddr:  255,
		TXAddr:  254,
		Mtu:     256,
	}
}

// TODO: Do we need to move the iso-tp connection into the Xport so it can be used for multiple sessions?

type ISOTPXport struct {
	cfg     *XportCfg
	started bool
}

func NewISOTPXport(cfg *XportCfg) *ISOTPXport {
	return &ISOTPXport{cfg: cfg}
}

func (ux *ISOTPXport) BuildSesn(cfg sesn.SesnCfg) (sesn.Sesn, error) {
	return NewISOTPSesn(ux.cfg, cfg)
}

func (ux *ISOTPXport) Start() error {
	if ux.started {
		return nmxutil.NewXportError("ISO-TP xport started twice")
	}
	ux.started = true
	return nil
}

func (ux *ISOTPXport) Stop() error {
	if !ux.started {
		return nmxutil.NewXportError("ISO-TP xport stopped twice")
	}
	ux.started = false
	return nil
}

func (ux *ISOTPXport) Tx(bytes []byte) error {
	return fmt.Errorf("unsupported")
}
