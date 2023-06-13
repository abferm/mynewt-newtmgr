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

package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/abferm/candi/isotp"
	"mynewt.apache.org/newt/util"
	"mynewt.apache.org/newtmgr/nmxact/nmisotp"
)

func einvalISOTPConnString(f string, args ...interface{}) error {
	suffix := fmt.Sprintf(f, args)
	return util.FmtNewtError("Invalid ISO-TP connstring; %s", suffix)
}

func ParseISOTPConnString(cs string) (*nmisotp.XportCfg, error) {
	sc := nmisotp.NewXportCfg()

	parts := strings.Split(cs, ",")
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		// Handle old-style conn string (single token indicating dev file).
		if len(kv) == 1 {
			kv = []string{"dev", kv[0]}
		}

		k := kv[0]
		v := kv[1]

		switch k {
		case "bus":
			sc.BusName = v

		case "receive":
			var rx, tx uint32
			_, err := fmt.Sscanf(v, "%d:%d", &rx, &tx)
			if err != nil {
				return sc, einvalISOTPConnString("Invalid Receive Address Pair: %s", v)
			}
			sc.ReceiveAddr = isotp.NewAddr(rx, tx)

		case "send":
			var rx, tx uint32
			_, err := fmt.Sscanf(v, "%d:%d", &rx, &tx)
			if err != nil {
				return sc, einvalISOTPConnString("Invalid Receive Address Pair: %s", v)
			}
			sc.SendAddr = isotp.NewAddr(rx, tx)

		case "mtu":
			var err error
			sc.Mtu, err = strconv.Atoi(v)
			if err != nil {
				return sc, einvalISOTPConnString("Invalid mtu: %s", v)
			}

		default:
			return sc, einvalISOTPConnString("Unrecognized key: %s", k)
		}
	}

	return sc, nil
}

func BuildISOTPXport(sc *nmisotp.XportCfg) (*nmisotp.ISOTPXport, error) {
	sx := nmisotp.NewISOTPXport(sc)
	if err := sx.Start(); err != nil {
		return nil, util.ChildNewtError(err)
	}

	return sx, nil
}
