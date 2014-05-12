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

package singleton

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fangli/gomsgfiber/logging"
	"github.com/fangli/gomsgfiber/recorder"
)

type SingletonProcessor struct {
	Name       string
	Retries    int
	RetryDelay time.Duration
	Command    string
	MsgChann   chan []byte
	Logger     *logging.Log
	Recorder   *recorder.Record
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

func removeTmpFile(fname string) {
	os.Remove(fname)
}

func (s *SingletonProcessor) ExecuteCommand(msg []byte) ([]byte, error) {
	fname, err := createTmpFile(s.Name, msg)
	if err != nil {
		s.Logger.Fatal("Unable creating temp file to run command: " + err.Error())
	}
	command := strings.Replace(s.Command, "%channel%", s.Name, -1)
	command = strings.Replace(command, "%file%", fname, -1)
	command = strings.Replace(command, "%content%", string(msg), -1)

	var execErr error
	var out []byte
	for i := 1; i <= s.Retries+1; i++ {
		s.Logger.Info("Executing #" + strconv.Itoa(i))
		out, execErr = exec.Command("sh", "-c", command).CombinedOutput()
		s.Logger.Info("Output: " + string(out))
		if execErr != nil {
			if i < s.Retries+1 {
				s.Logger.Warning("Failed to execute command, will retry after " + s.RetryDelay.String())
				time.Sleep(s.RetryDelay)
			} else {
				s.Logger.Error("All retries failed at last, aborted!")
			}
		} else {
			break
		}
	}
	removeTmpFile(fname)

	return out, execErr
}

func (s *SingletonProcessor) Run() {
	var err error
	for body := range s.MsgChann {
		if !s.Recorder.Equal(s.Name, body) {
			s.Logger.Info("Received new command from server, creating subtask...")
			_, err = s.ExecuteCommand(body)
			if err != nil {
				s.Logger.Error("An error occured when running command. " + err.Error())
			} else {
				s.Logger.Info("Command finished and exit")
			}
			s.Recorder.Set(s.Name, body)
		}
	}
}
