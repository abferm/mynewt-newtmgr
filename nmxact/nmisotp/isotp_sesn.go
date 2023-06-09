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
	"net"
	"time"

	"github.com/abferm/candi/isotp"
	"github.com/runtimeco/go-coap"
	"mynewt.apache.org/newtmgr/nmxact/mgmt"
	"mynewt.apache.org/newtmgr/nmxact/nmcoap"
	"mynewt.apache.org/newtmgr/nmxact/nmp"
	"mynewt.apache.org/newtmgr/nmxact/nmxutil"
	"mynewt.apache.org/newtmgr/nmxact/omp"
	"mynewt.apache.org/newtmgr/nmxact/sesn"
)

type ISOTPSesn struct {
	cfg   sesn.SesnCfg
	xpCfg *XportCfg
	conn  net.Conn
	txvr  *mgmt.Transceiver
}

func NewISOTPSesn(xpCfg *XportCfg, cfg sesn.SesnCfg) (*ISOTPSesn, error) {
	s := &ISOTPSesn{
		cfg:   cfg,
		xpCfg: xpCfg,
	}
	txvr, err := mgmt.NewTransceiver(cfg.TxFilter, cfg.RxFilter, false,
		cfg.MgmtProto, 3)
	if err != nil {
		return nil, err
	}
	s.txvr = txvr

	return s, nil
}

func (s *ISOTPSesn) Open() error {
	if s.conn != nil {
		return nmxutil.NewSesnAlreadyOpenError(
			"Attempt to open an already-open ISO-TP session")
	}

	bus, err := isotp.BusByName(s.xpCfg.BusName)
	if err != nil {
		return err
	}

	conn, err := bus.Dial(isotp.NewAddr(s.xpCfg.RXAddr, s.xpCfg.TXAddr))
	if err != nil {
		return err
	}
	s.conn = conn

	return nil
}

func (s *ISOTPSesn) Close() error {
	if s.conn == nil {
		return nmxutil.NewSesnClosedError(
			"Attempt to close an unopened ISOTP session")
	}

	s.conn.Close()
	s.txvr.ErrorAll(fmt.Errorf("closed"))
	s.txvr.Stop()
	s.conn = nil
	return nil
}

func (s *ISOTPSesn) IsOpen() bool {
	return s.conn != nil
}

func (s *ISOTPSesn) MtuIn() int {
	return s.xpCfg.Mtu - omp.OMP_MSG_OVERHEAD
}

func (s *ISOTPSesn) MtuOut() int {
	return s.xpCfg.Mtu - omp.OMP_MSG_OVERHEAD
}

func (s *ISOTPSesn) TxRxMgmt(m *nmp.NmpMsg,
	timeout time.Duration) (nmp.NmpRsp, error) {

	if !s.IsOpen() {
		return nil, fmt.Errorf("Attempt to transmit over closed ISO-TP session")
	}

	txRaw := func(b []byte) error {
		_, err := s.conn.Write(b)
		return err
	}
	return s.txvr.TxRxMgmt(txRaw, m, s.MtuOut(), timeout)
}

func (s *ISOTPSesn) TxRxMgmtAsync(m *nmp.NmpMsg,
	timeout time.Duration, ch chan nmp.NmpRsp, errc chan error) error {
	rsp, err := s.TxRxMgmt(m, timeout)
	if err != nil {
		errc <- err
	} else {
		ch <- rsp
	}
	return nil
}

func (s *ISOTPSesn) AbortRx(seq uint8) error {
	s.txvr.ErrorAll(fmt.Errorf("Rx aborted"))
	return nil
}

func (s *ISOTPSesn) TxCoap(m coap.Message) error {

	if !s.IsOpen() {
		return fmt.Errorf("Attempt to transmit over closed ISO-TP session")
	}
	txRaw := func(b []byte) error {
		_, err := s.conn.Write(b)
		return err
	}

	return s.txvr.TxCoap(txRaw, m, s.MtuOut())
}

func (s *ISOTPSesn) MgmtProto() sesn.MgmtProto {
	return s.cfg.MgmtProto
}

func (s *ISOTPSesn) ListenCoap(mc nmcoap.MsgCriteria) (*nmcoap.Listener, error) {
	return s.txvr.ListenCoap(mc)
}

func (s *ISOTPSesn) StopListenCoap(mc nmcoap.MsgCriteria) {
	s.txvr.StopListenCoap(mc)
}

func (s *ISOTPSesn) CoapIsTcp() bool {
	return false
}

func (s *ISOTPSesn) RxAccept() (sesn.Sesn, *sesn.SesnCfg, error) {
	return nil, nil, fmt.Errorf("Op not implemented yet")
}

func (s *ISOTPSesn) RxCoap(opt sesn.TxOptions) (coap.Message, error) {
	return nil, fmt.Errorf("Op not implemented yet")
}

func (s *ISOTPSesn) Filters() (nmcoap.TxMsgFilter, nmcoap.RxMsgFilter) {
	return s.txvr.Filters()
}

func (s *ISOTPSesn) SetFilters(txFilter nmcoap.TxMsgFilter,
	rxFilter nmcoap.RxMsgFilter) {

	s.txvr.SetFilters(txFilter, rxFilter)
}
