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
	"github.com/fangli/gomsgfiber/executer/singleton"
	"github.com/fangli/gomsgfiber/logging"
	"github.com/fangli/gomsgfiber/parsecfg"
	"github.com/fangli/gomsgfiber/recorder"
)

type Msg struct {
	Channel string
	Message []byte
}

func Run(config *parsecfg.Config, msgChann *chan Msg) {
	var msg Msg
	// Make a cache query instance
	recorder := recorder.Record{config.Main.Data_Path + "/.caches"}

	// Pool contains different message channel
	pool := make(map[string]*singleton.SingletonProcessor)

	for chnName, chnConfig := range config.Channel.Channels {

		pool[chnName] = &singleton.SingletonProcessor{
			Name:       chnName,
			Retries:    chnConfig.Retry_Times,
			RetryDelay: chnConfig.Retry_Delay,
			Command:    chnConfig.Command,
			MsgChann:   make(chan []byte, config.Main.Queue_Size),
			Logger: &logging.Log{
				Dest:     1,
				Level:    config.Main.Log_Level,
				FileName: config.Main.Data_Path + "/logs/" + chnName + ".log",
			},
			Recorder: &recorder,
		}

		go pool[chnName].Run()
	}

	for msg = range *msgChann {
		pool[msg.Channel].MsgChann <- msg.Message
	}
}
