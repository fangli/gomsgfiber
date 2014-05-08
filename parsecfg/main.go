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

package parsecfg

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fangli/gomsgfiber/logging"

	"code.google.com/p/gcfg"
)

var SYS_VER string
var SYS_BUILD_VER string
var SYS_BUILD_DATE string

type Config struct {
	Main struct {
		Data_Path     string
		Queue_Size    int64
		Raw_Log_Level string `gcfg:"log-level"`
		Log_Level     int    `gcfg:"raw-log-level"`
	}
	Client struct {
		Raw_Nodes         string `gcfg:"msgfiber-nodes"`
		Nodes             []string
		Psk               string
		Raw_Ping_Interval string `gcfg:"heartbeat-interval"`
		Ping_Interval     time.Duration
	}
	Channel struct {
		Include      string
		Channels     map[string]*dryChannel
		Channel_List []string
	}
	AppLog *logging.Log
}

type dryChannel struct {
	Command     string
	Retry_Times int64
}

type rawChannelConfig struct {
	Channel map[string]*dryChannel
}

func Parse() Config {
	configFile := flag.String("config", "/etc/msgclient/msgclient.conf", "The primary configuration file")
	version := flag.Bool("version", false, "Show version information")
	v := flag.Bool("v", false, "Show version information")

	flag.Parse()

	if *version || *v {
		fmt.Println("Msgclient: A client of msgfiber")
		fmt.Println("Version", SYS_VER)
		fmt.Println("Build", SYS_BUILD_VER)
		fmt.Println("Compile at", SYS_BUILD_DATE)
		os.Exit(0)
	}

	var config Config
	err := gcfg.ReadFileInto(&config, *configFile)
	if err != nil {
		log.Fatalf("Failed to parse config data: %s", err)
	}

	channelFileList, err := filepath.Glob(config.Channel.Include)
	if err != nil {
		log.Fatal(err.Error())
	}

	if channelFileList == nil {
		log.Fatal("No channels subscribed, please check the configuration folder ", config.Channel.Include)
	}

	channelConfig := make(map[string]*dryChannel)

	for _, fname := range channelFileList {
		var _rawChannels rawChannelConfig
		err = gcfg.ReadFileInto(&_rawChannels, fname)
		if err == nil {
			for name, chann := range _rawChannels.Channel {
				channelConfig[name] = &dryChannel{}
				channelConfig[name].Command = chann.Command
				channelConfig[name].Retry_Times = chann.Retry_Times
			}
		}
	}

	for chname, _ := range channelConfig {
		config.Channel.Channel_List = append(config.Channel.Channel_List, chname)
	}

	config.Channel.Channels = channelConfig
	config.Client.Nodes = strings.Split(config.Client.Raw_Nodes, ",")

	syncDuration, err := time.ParseDuration(config.Client.Raw_Ping_Interval)
	if err != nil {
		log.Fatal("Config ping-interval is not acceptable: ", config.Client.Raw_Ping_Interval)
	}
	config.Client.Ping_Interval = syncDuration

	config.Main.Data_Path = strings.TrimRight(config.Main.Data_Path, "/")

	config.Main.Log_Level = logging.LevelInt[strings.ToUpper(config.Main.Raw_Log_Level)]

	config.AppLog = &logging.Log{
		Dest:     logging.FILE,
		Level:    config.Main.Log_Level,
		FileName: config.Main.Data_Path + "/application.log",
	}

	return config
}
