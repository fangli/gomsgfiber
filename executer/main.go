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

package executer

import (
	"os"

	"github.com/fangli/gomsgfiber/executer/singleton"
	"github.com/fangli/gomsgfiber/logging"
	"github.com/fangli/gomsgfiber/parsecfg"
	"github.com/fangli/gomsgfiber/recorder"
)

type Msg struct {
	Channel string
	Message []byte
}

func RunForever(config *parsecfg.Config, msgChann *chan Msg) {
	var msg Msg
	// Make a cache query instance

	// Pool contains different message channel
	pool := make(map[string]*singleton.SingletonProcessor)

	for chnName, chnConfig := range config.Channel.Channels {
		pool[chnName] = &singleton.SingletonProcessor{
			ChnName:      chnName,
			Retries:      chnConfig.Retry_Times,
			RetryDelay:   chnConfig.Retry_Delay,
			UploadOutput: chnConfig.Upload_Output,
			Command:      chnConfig.Command,
			MsgChann:     make(chan []byte, config.Main.Queue_Size),
			ReportUrl:    config.Main.Report_Url,
			InstanceName: config.Client.Name,
			InstanceId:   config.Client.Instance_Id,
			Recorder:     &recorder.Record{config.Main.Data_Path + "/.caches/" + chnName},
			Logger: &logging.Log{
				Dest:     1,
				Level:    config.Main.Log_Level,
				FileName: config.Main.Data_Path + "/logs/" + chnName + ".log",
			},
		}
		go pool[chnName].Run()
	}

	for msg = range *msgChann {
		pool[msg.Channel].MsgChann <- msg.Message
	}
}

func RunOnce(config *parsecfg.Config, msgChann *chan Msg) {
	var msg Msg
	// Make a cache query instance

	// Pool contains different message channel
	pool := make(map[string]*singleton.SingletonProcessor)

	for chnName, chnConfig := range config.Channel.Channels {
		pool[chnName] = &singleton.SingletonProcessor{
			ChnName:      chnName,
			Retries:      chnConfig.Retry_Times,
			RetryDelay:   chnConfig.Retry_Delay,
			UploadOutput: chnConfig.Upload_Output,
			Command:      chnConfig.Command,
			MsgChann:     make(chan []byte, config.Main.Queue_Size),
			ReportUrl:    config.Main.Report_Url,
			InstanceName: config.Client.Name,
			InstanceId:   config.Client.Instance_Id,
			Recorder:     &recorder.Record{config.Main.Data_Path + "/.caches/" + chnName},
			Logger: &logging.Log{
				Dest:     1,
				Level:    config.Main.Log_Level,
				FileName: config.Main.Data_Path + "/logs/" + chnName + ".log",
			},
		}
	}

	go func() {
		for msg = range *msgChann {
			pool[msg.Channel].MsgChann <- msg.Message
		}
	}()

	outputChan := make(chan error, len(config.Channel.Channels))
	for chnName, _ := range config.Channel.Channels {
		go func(cn string) {
			outputChan <- pool[cn].RunOnce()
		}(chnName)
	}

	var err error = nil
	for _, _ = range config.Channel.Channels {
		e := <-outputChan
		if e != nil {
			err = e
		}
	}

	if err == nil {
		os.Exit(0)
	} else {
		os.Exit(1)
	}

}

func Run(config *parsecfg.Config, msgChann *chan Msg) {
	if config.IsInitial == true {
		RunOnce(config, msgChann)
	} else {
		RunForever(config, msgChann)
	}
}
