/*************************************************************************
* This file is a part of msgfiber, A decentralized and distributed message
* synchronization system

* Copyright (C) 2014  Fang Li <surivlee@gmail.com> and Funplus, Inc.
*
* This program is free software; you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation; either version 2 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License along
* with this program; if not, see http://www.gnu.org/licenses/gpl-2.0.html
*************************************************************************/

package nodetest

import (
	"errors"
	"net"
	"time"

	"github.com/vmihailenco/msgpack"
)

type Result struct {
	Name string
	err  error
}

// Connection speed and delay test
type Tester struct {
	Nodes []string
}

func (c *Tester) checkNode(node string, resultChn chan Result) {
	cmd := make(map[string]interface{})
	cmd["Op"] = "p"
	payload, _ := msgpack.Marshal(cmd)

	conn, err := net.DialTimeout("tcp", node, time.Second*3)
	if err != nil {
		resultChn <- Result{node, err}
		return
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	_, err = conn.Write(payload)
	if err != nil {
		resultChn <- Result{node, err}
		return
	}

	var resp struct {
		Op string
	}
	decoder := msgpack.NewDecoder(conn)
	conn.SetReadDeadline(time.Now().Add(time.Second * 3))
	err = decoder.Decode(&resp)
	if err != nil {
		resultChn <- Result{node, err}
		return
	}

	if resp.Op == "p" {
		resultChn <- Result{node, nil}
	} else {
		resultChn <- Result{node, errors.New("Unknown error")}
	}
}

func (c *Tester) Run() ([]Result, error) {
	resultChn := make(chan Result, len(c.Nodes))
	var result []Result

	for _, node := range c.Nodes {
		go c.checkNode(node, resultChn)
	}

	for _, _ = range c.Nodes {
		ret := <-resultChn
		if ret.err == nil {
			result = append(result, ret)
		}
	}
	close(resultChn)

	if len(result) == 0 {
		return nil, errors.New("No valid candidate servers found")
	}

	return result, nil
}
