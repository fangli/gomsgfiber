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
	"io"
	"net"
	"time"

	"github.com/fangli/gomsgfiber/executer"
	"github.com/fangli/gomsgfiber/nodetest"
	"github.com/fangli/gomsgfiber/parsecfg"
	"github.com/fangli/gomsgfiber/processor/connmanager"
)

// gomsgfiber Client package

type Processor struct {
	Config     *parsecfg.Config
	tester     *nodetest.Tester
	MsgChannel chan executer.Msg
}

func (p *Processor) RunConnManager() error {
	var err error
	// Ping test and choose the fastest connection
	p.Config.AppLog.Info("Check connections and deciding the best msgfiber server...")
	nodes, err := p.tester.Run()
	if err != nil {
		return err
	}

	// Dial up to best node
	p.Config.AppLog.Info("The fastest server is " + nodes[0].Name + ", now connecting...")
	conn, err := net.DialTimeout("tcp", nodes[0].Name, time.Second*3)
	if err != nil {
		return err
	}
	p.Config.AppLog.Info("Connected to node " + nodes[0].Name)

	// Prepare the msg channel
	node := connmanager.Manager{
		Conn:         conn,
		Psk:          []byte(p.Config.Client.Psk),
		Channel:      p.Config.Channel.Channel_List,
		PingInterval: p.Config.Client.Ping_Interval,
	}

	err = node.Subscribe()
	if err != nil {
		return err
	}

	go node.Heartbeat()
	err = node.Receiver(&p.MsgChannel)
	return err
}

func (p *Processor) Forever() {
	for {
		err := p.RunConnManager()
		switch err {
		case io.EOF:
			p.Config.AppLog.Error("Connection closed by msgfiber server")
		default:
			p.Config.AppLog.Error("Communication Error: " + err.Error())
		}
		time.Sleep(time.Second)
	}
}

func (p *Processor) Init() {
	p.Config.AppLog.Info("Msgclient started...")
	p.tester = &nodetest.Tester{
		Nodes: p.Config.Client.Nodes,
	}
	p.MsgChannel = make(chan executer.Msg, p.Config.Main.Queue_Size)

	p.Config.AppLog.Info("Initialization...")
	go executer.Run(p.Config, &p.MsgChannel)
}
