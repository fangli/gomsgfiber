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
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fangli/gomsgfiber/logging"
	"github.com/fangli/gomsgfiber/recorder"
	"github.com/mreiferson/go-httpclient"
)

type SingletonProcessor struct {
	ChnName      string
	Retries      int
	RetryDelay   time.Duration
	Command      string
	MsgChann     chan []byte
	Logger       *logging.Log
	Recorder     *recorder.Record
	ReportUrl    string
	InstanceName string
	InstanceId   string
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
	fname, err := createTmpFile(s.ChnName, msg)
	if err != nil {
		s.Logger.Fatal("Unable creating temp file to run command: " + err.Error())
	}
	command := strings.Replace(s.Command, "%channel%", s.ChnName, -1)
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

func (s *SingletonProcessor) fetchInfo() {
	data := url.Values{}
	data.Set("action", "polling")
	data.Add("instance_id", s.InstanceId)
	data.Add("channel", s.ChnName)
	data.Add("release_id", strconv.FormatInt(s.Recorder.GetReleaseId(), 10))
	data.Add("name", s.InstanceName)

	transport := &httpclient.Transport{
		ConnectTimeout:        5 * time.Second,
		RequestTimeout:        5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	defer transport.Close()

	client := &http.Client{Transport: transport}
	req, _ := http.NewRequest("POST", s.ReportUrl, bytes.NewBufferString(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	resp, _ := client.Do(req)
	defer resp.Body.Close()
}

func (s *SingletonProcessor) heartbeat() {
	for {
		s.fetchInfo()
		time.Sleep(2 * time.Second)
	}
}

func (s *SingletonProcessor) Run() {
	var err error
	go s.heartbeat()

	for body := range s.MsgChann {
		if !s.Recorder.Equal(body) {
			s.Logger.Info("Received new command from server, creating subtask...")
			_, err = s.ExecuteCommand(body)
			if err != nil {
				s.Logger.Error("An error occured when running command. " + err.Error())
			} else {
				s.Logger.Info("Command finished and exit")
			}
			s.Recorder.Set(body)
		}
	}
}
