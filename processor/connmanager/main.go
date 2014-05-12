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

package connmanager

import (
	"net"
	"time"

	"github.com/fangli/gomsgfiber/executer"
	"github.com/vmihailenco/msgpack"
)

type Manager struct {
	Conn         net.Conn
	Psk          []byte
	Channel      []string
	PingInterval time.Duration
}

func (m *Manager) Exit() {
	m.Conn.Close()
}

func (m *Manager) Subscribe() error {
	subStruct := make(map[string]interface{})
	subStruct["Psk"] = m.Psk
	subStruct["Op"] = "subscribe"
	subStruct["Channel"] = m.Channel
	subPack, _ := msgpack.Marshal(subStruct)
	m.Conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	_, err := m.Conn.Write(subPack)
	return err
}

func (m *Manager) Heartbeat() {
	pingStruct := make(map[string]interface{})
	pingStruct["Op"] = "p"
	pingPacket, _ := msgpack.Marshal(pingStruct)

	for {
		time.Sleep(m.PingInterval)
		m.Conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
		_, err := m.Conn.Write(pingPacket)
		if err != nil {
			return
		}
	}
}

func (m *Manager) Receiver(msgChan *chan executer.Msg) error {
	defer m.Exit()
	decoder := msgpack.NewDecoder(m.Conn)
	var raw map[string]interface{}
	var err error
	for {
		m.Conn.SetReadDeadline(time.Now().Add(m.PingInterval * 3))
		err = decoder.Decode(&raw)
		if err != nil {
			return err
		}
		if raw["Op"] == "sync" {
			*msgChan <- executer.Msg{
				Channel: raw["Channel"].(string),
				Message: []byte(raw["Message"].(string)),
			}
		} else if raw["Op"] == "subscribe" {
			for k, v := range raw["Channel"].(map[interface{}]interface{}) {
				*msgChan <- executer.Msg{
					Channel: k.(string),
					Message: []byte(v.(string)),
				}
			}
		}
	}
}
