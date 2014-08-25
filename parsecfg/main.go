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
	"errors"
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

type ChannelStruct struct {
	Command         string
	Retry_Times     int
	Upload_Output   bool          `gcfg:"upload-output"`
	Raw_Retry_Delay string        `gcfg:"retry-delay"`
	Retry_Delay     time.Duration `gcfg:"raw-retry-delay"`
}

type ChannelsMap struct {
	Channel map[string]*ChannelStruct
}

type Config struct {
	Manager struct {
		Command string
	}
	Main struct {
		Data_Path     string
		Queue_Size    int64
		Raw_Log_Level string `gcfg:"log-level"`
		Log_Level     int    `gcfg:"raw-log-level"`
		Report_Url    string `gcfg:"report-url"`
	}
	Client struct {
		Name              string `gcfg:"name"`
		Instance_Id       string `gcfg:"instance-id"`
		Raw_Nodes         string `gcfg:"msgfiber-nodes"`
		Nodes             []string
		Psk               string
		Raw_Ping_Interval string `gcfg:"heartbeat-interval"`
		Ping_Interval     time.Duration
	}
	Channel struct {
		Include      string
		Channels     map[string]*ChannelStruct
		Channel_List []string
	}
	AppLog    *logging.Log
	PidFile   string
	IsInitial bool
}

func mkdirs(path string) error {
	f, err := os.Stat(path)
	if err != nil {
		if os.MkdirAll(path, os.ModePerm) != nil {
			return errors.New("Unable to initialize dir: " + path)
		}
	}

	f, err = os.Stat(path)
	if !f.IsDir() {
		return errors.New(path + " must be a directory")
	}
	return nil
}

func initialDataDir(datadir string) error {

	// Initial all folders
	var err error
	if err = mkdirs(datadir); err != nil {
		return err
	}
	if err = mkdirs(datadir + "/logs"); err != nil {
		return err
	}
	if err = mkdirs(datadir + "/.caches"); err != nil {
		return err
	}
	return nil
}

func showVersion() {
	fmt.Println("Msgclient: A decentralized and distributed message synchronization system")
	fmt.Println("Version", SYS_VER)
	fmt.Println("Build", SYS_BUILD_VER)
	fmt.Println("Compile at", SYS_BUILD_DATE)
	os.Exit(0)
}

func getConfigPath() (string, string, bool) {
	configPath := flag.String("config", "/etc/msgclient/msgclient.conf", "The primary configuration file")
	pidfile := flag.String("pid", "/var/run/msgclient.pid", "The PID file of msgclient")
	initial := flag.Bool("initial", false, "Initialize all scripts and exit. Usually used in the first run")
	v := flag.Bool("v", false, "Show version information")
	flag.Parse()

	if *v {
		showVersion()
	}
	return *configPath, *pidfile, *initial
}

func initialDefault() *Config {
	config := new(Config)
	config.Main.Data_Path = "/var/lib/msgclient"
	config.Main.Queue_Size = 100
	config.Main.Raw_Log_Level = "INFO"
	config.Main.Log_Level = logging.INFO
	config.Main.Report_Url = "http://localhost/status_callback.php"
	config.Client.Name = "N/A"
	config.Client.Instance_Id = "i-000000"
	config.Client.Raw_Nodes = "localhost:3264"
	config.Client.Nodes = []string{"localhost:3264"}
	config.Client.Psk = ""
	config.Client.Raw_Ping_Interval = "1s"
	config.Client.Ping_Interval = time.Second
	config.Channel.Include = "/etc/msgclient/conf.d"
	return config
}

func getChannels(flist []string) *map[string]*ChannelStruct {
	var err error
	ret := make(map[string]*ChannelStruct)

	for _, fname := range flist {
		var _rawChannels ChannelsMap
		err = gcfg.ReadFileInto(&_rawChannels, fname)
		if err != nil {
			log.Fatalf("Cann't load channel config from %s", fname)
		}
		for name, chann := range _rawChannels.Channel {
			ret[name] = &ChannelStruct{}
			ret[name].Command = chann.Command
			ret[name].Retry_Times = chann.Retry_Times
			ret[name].Upload_Output = chann.Upload_Output

			if chann.Raw_Retry_Delay == "" {
				ret[name].Retry_Delay = time.Duration(0)
			} else {
				ret[name].Retry_Delay, err = time.ParseDuration(chann.Raw_Retry_Delay)
				if err != nil {
					log.Fatalf("Unable to parse config retry-delay in file %s!", fname)
				}
			}
		}
	}
	return &ret
}

func Parse() *Config {

	configPath, pidfile, initial := getConfigPath()

	config := initialDefault()
	err := gcfg.ReadFileInto(config, configPath)
	if err != nil {
		log.Fatalf("Failed to read config from %s, Reason: %s", configPath, err)
	}

	subConfigfiles, err := filepath.Glob(config.Channel.Include)
	if err != nil {
		log.Fatalf("Unable reading channel config files by %s, Reason: %s", config.Channel.Include, err)
	}
	if subConfigfiles == nil {
		log.Fatalf("No channel config file found at %s, you must subscribe at least one channel!", config.Channel.Include)
	}

	channelConfig := getChannels(subConfigfiles)

	for chname, _ := range *channelConfig {
		config.Channel.Channel_List = append(config.Channel.Channel_List, chname)
	}

	config.Channel.Channels = *channelConfig
	config.Client.Nodes = strings.Split(config.Client.Raw_Nodes, ",")

	syncDuration, err := time.ParseDuration(config.Client.Raw_Ping_Interval)
	if err != nil {
		log.Fatalf("Config ping-interval is not acceptable")
	}
	config.Client.Ping_Interval = syncDuration

	config.Main.Data_Path = strings.TrimRight(config.Main.Data_Path, "/")
	config.Main.Log_Level = logging.LevelInt[strings.ToUpper(config.Main.Raw_Log_Level)]

	err = initialDataDir(config.Main.Data_Path)
	if err != nil {
		log.Fatalf("Unable initializing data dir: " + config.Main.Data_Path + " (" + err.Error() + ")")
	}

	config.AppLog = &logging.Log{
		Dest:     logging.FILE,
		Level:    config.Main.Log_Level,
		FileName: config.Main.Data_Path + "/application.log",
	}

	config.PidFile = pidfile
	config.IsInitial = initial

	return config
}
