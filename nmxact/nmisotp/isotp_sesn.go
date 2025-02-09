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
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/abferm/candi/isotp"
	"github.com/runtimeco/go-coap"
	log "github.com/sirupsen/logrus"
	"mynewt.apache.org/newtmgr/nmxact/mgmt"
	"mynewt.apache.org/newtmgr/nmxact/nmcoap"
	"mynewt.apache.org/newtmgr/nmxact/nmp"
	"mynewt.apache.org/newtmgr/nmxact/nmxutil"
	"mynewt.apache.org/newtmgr/nmxact/omp"
	"mynewt.apache.org/newtmgr/nmxact/sesn"
	"mynewt.apache.org/newtmgr/nmxact/udp"
)

type ISOTPSesn struct {
	cfg                   sesn.SesnCfg
	xpCfg                 *XportCfg
	sendConn, receiveConn net.Conn
	txvr                  *mgmt.Transceiver
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
	if s.sendConn != nil {
		return nmxutil.NewSesnAlreadyOpenError(
			"Attempt to open an already-open ISO-TP session")
	}

	bus, err := isotp.BusByName(s.xpCfg.BusName)
	if err != nil {
		return err
	}

	send, err := bus.Dial(s.xpCfg.SendAddr)
	if err != nil {
		return err
	}
	s.sendConn = send

	receive, err := bus.Dial(s.xpCfg.ReceiveAddr)
	if err != nil {
		return err
	}
	s.receiveConn = receive

	go func() {
		for {
			err := s.rx()
			if errors.Is(err, os.ErrClosed) {
				return
			}
		}
	}()

	return nil
}

func (s *ISOTPSesn) rx() error {
	// NOTE: there have been issues trying to read just the mtu size,
	// read in MAX_PACKET_SIZE instead like the udp transport does
	buff := make([]byte, udp.MAX_PACKET_SIZE)
	rxLen, err := s.receiveConn.Read(buff)
	if err != nil {
		s.txvr.ErrorAll(fmt.Errorf("RX Error: %w", err))
		log.Errorf("ISO-TP RX Error: %s", err)
		return err
	}
	log.Debugf("ISO-TP Read %d bytes", rxLen)
	if s.cfg.MgmtProto == sesn.MGMT_PROTO_OMP {
		s.txvr.DispatchCoap(buff[:rxLen])
	} else if s.cfg.MgmtProto == sesn.MGMT_PROTO_NMP {
		s.txvr.DispatchNmpRsp(buff[:rxLen])
	}
	return nil
}

func (s *ISOTPSesn) Close() error {
	if !s.IsOpen() {
		return nmxutil.NewSesnClosedError(
			"Attempt to close an unopened ISOTP session")
	}

	s.sendConn.Close()
	s.receiveConn.Close()
	s.txvr.ErrorAll(fmt.Errorf("closed"))
	s.txvr.Stop()
	s.sendConn = nil
	s.receiveConn = nil
	return nil
}

func (s *ISOTPSesn) IsOpen() bool {
	return s.sendConn != nil && s.receiveConn != nil
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
		_, err := s.sendConn.Write(b)
		if err != nil {
			return fmt.Errorf("TX Error: %w", err)
		}
		return nil
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
		_, err := s.sendConn.Write(b)
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
