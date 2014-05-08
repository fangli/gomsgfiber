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
*`
* You should have received a copy of the GNU General Public License along
* with this program; if not, see http://www.gnu.org/licenses/gpl-2.0.html
*************************************************************************/

package processor

import (
	"log"
	"net"
	"time"

	"github.com/fangli/gomsgfiber/executer"
	"github.com/fangli/gomsgfiber/nodetest"
	"github.com/fangli/gomsgfiber/parsecfg"
	"github.com/vmihailenco/msgpack"
)

// gomsgfiber Client package

type generalStruct map[string]interface{}

type cmdStruct struct {
	Op string
}

type syncStruct struct {
	Channel string
	Message []byte
}

type getStruct struct {
	Channel map[string][]byte
}

type nodeConn struct {
	Conn         net.Conn
	Psk          []byte
	Channel      []string
	pingInterval time.Duration
}

func (n *nodeConn) subscribe() {
	subStruct := make(map[string]interface{})
	subStruct["Psk"] = n.Psk
	subStruct["Op"] = "subscribe"
	subStruct["Channel"] = n.Channel
	subPack, _ := msgpack.Marshal(subStruct)
	n.Conn.Write(subPack)
}

func ExitNotification(msg string) {
	log.Println(msg)
}

func (n *nodeConn) heartbeat() {
	pingStruct := make(map[string]interface{})
	pingStruct["Op"] = "p"
	pingPacket, _ := msgpack.Marshal(pingStruct)

	for {
		time.Sleep(n.pingInterval)
		n.Conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
		_, err := n.Conn.Write(pingPacket)
		if err != nil {
			return
		}
	}
}

func (n *nodeConn) Receiver(msgChan *chan executer.Msg) {
	defer ExitNotification("subscriber exited")
	decoder := msgpack.NewDecoder(n.Conn)
	var raw generalStruct
	var err error
	for {
		n.Conn.SetReadDeadline(time.Now().Add(n.pingInterval * 3))
		err = decoder.Decode(&raw)
		if err != nil {
			log.Println(err.Error())
			return
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

type Processor struct {
	Config     parsecfg.Config
	tester     nodetest.Tester
	MsgChannel chan executer.Msg
}

func (p *Processor) singleWorker() error {
	// Ping test and choose the fastest connection
	testResult, err := p.tester.Run()
	if err != nil {
		return err
	}

	remote := testResult[0].Name

	// Dial up to best node
	log.Println("Connecting to", remote)
	conn, err := net.DialTimeout("tcp", remote, time.Second*3)
	if err != nil {
		return err
	}
	log.Println(remote, "connected")

	// Prepare the msg channel

	node := nodeConn{
		Conn:         conn,
		Psk:          []byte(p.Config.Client.Psk),
		Channel:      p.Config.Channel.Channel_List,
		pingInterval: p.Config.Client.Ping_Interval,
	}

	node.subscribe()
	log.Println("Subscribed")
	go node.heartbeat()
	node.Receiver(&p.MsgChannel)
	return nil
}

func (p *Processor) Forever() {
	// for msg := range p.MsgChannel {
	// 	log.Printf("%s", msg)
	// }
	for {
		p.singleWorker()
	}
}

func (p *Processor) Init() {
	p.tester = nodetest.Tester{
		Nodes: p.Config.Client.Nodes,
	}
	p.MsgChannel = make(chan executer.Msg, p.Config.Main.Queue_Size)

	go executer.Run(p.Config, &p.MsgChannel)
}
