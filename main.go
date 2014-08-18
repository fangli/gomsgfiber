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

package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/fangli/gomsgfiber/parsecfg"
	"github.com/fangli/gomsgfiber/processor"
)

// Application framework
func WritePid(pidfile string) {
	err := ioutil.WriteFile(pidfile, []byte(strconv.Itoa(os.Getpid())), 0666)
	if err != nil {
		log.Fatalln("Error writing PID file: " + err.Error())
	}
}

func RemovePid(pidfile string) {
	err := os.Remove(pidfile)
	if err != nil {
		log.Println("System exit but unable to delete PID file: " + err.Error())
	}
}

func signalStopListener(config *parsecfg.Config, signalChn chan os.Signal) {
	<-signalChn
	RemovePid(config.PidFile)
	config.AppLog.Info("Shutdown signal received, msgclient stopped")
	os.Exit(0)
}

func signalProcessor(config *parsecfg.Config) {
	WritePid(config.PidFile)
	config.AppLog.Info("System started with PID " + strconv.Itoa(os.Getpid()) + "...")
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	go signalStopListener(config, signalChannel)
}

// Main
func main() {
	config := parsecfg.Parse()
	signalProcessor(config)
	p := processor.Processor{Config: config}
	p.Init()
	p.Forever()
}
