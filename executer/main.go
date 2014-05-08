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
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/fangli/gomsgfiber/logging"
	"github.com/fangli/gomsgfiber/parsecfg"
	"github.com/fangli/gomsgfiber/recorder"
)

type Msg struct {
	Channel string
	Message []byte
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

func createTmpFile(prefix string, data []byte) (string, error) {
	f, err := ioutil.TempFile("", prefix+"-")
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		return "", err
	}
	return f.Name(), nil
}

type SingletonProcessor struct {
	Name     string
	Retries  int64
	Command  string
	MsgChann chan []byte
	Logger   *logging.Log
	Recorder *recorder.Record
}

func (s *SingletonProcessor) ExecuteCommand(msg []byte) ([]byte, error) {
	fname, err := createTmpFile(s.Name, msg)
	if err != nil {
		log.Fatal("Unable creating temp file: " + err.Error())
	}
	command := strings.Replace(s.Command, "%channel%", s.Name, -1)
	command = strings.Replace(command, "%file%", fname, -1)

	out, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		return []byte(""), err
	}
	return out, nil
}

func (s *SingletonProcessor) Run() {
	for body := range s.MsgChann {
		if !s.Recorder.Equal(s.Name, body) {
			output, err := s.ExecuteCommand(body)
			if err != nil {
				s.Logger.Error(err.Error())
			} else {
				s.Logger.Info(string(output))
				s.Recorder.Set(s.Name, body)
			}
		}
	}
}

func Run(config parsecfg.Config, msgChann *chan Msg) {
	err := initialDataDir(config.Main.Data_Path)
	if err != nil {
		config.AppLog.Fatal(err.Error())
	}

	pool := make(map[string]*SingletonProcessor)
	recorder := recorder.Record{config.Main.Data_Path + "/.caches"}

	for chnName, chnConfig := range config.Channel.Channels {

		pool[chnName] = &SingletonProcessor{
			Name:     chnName,
			Retries:  chnConfig.Retry_Times,
			Command:  chnConfig.Command,
			MsgChann: make(chan []byte, config.Main.Queue_Size),
			Logger: &logging.Log{
				Dest:     1,
				Level:    config.Main.Log_Level,
				FileName: config.Main.Data_Path + "/logs/" + chnName + ".log",
			},
			Recorder: &recorder,
		}

		go pool[chnName].Run()
	}

	for msg := range *msgChann {
		pool[msg.Channel].MsgChann <- msg.Message
	}
}
