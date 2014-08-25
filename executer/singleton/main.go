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
	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
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
	UploadOutput bool
	MsgChann     chan []byte
	Logger       *logging.Log
	Recorder     *recorder.Record
	ReportUrl    string
	InstanceName string
	InstanceId   string
}

func parseBody(body []byte) (*recorder.Msgbody, error) {
	msgbody := recorder.Msgbody{}
	err := json.Unmarshal(body, &msgbody)
	if err != nil {
		return nil, err
	}
	return &msgbody, nil
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
	f.Chmod(0777)
	return f.Name(), nil
}

func removeTmpFile(fname string) {
	os.Remove(fname)
}

func (s *SingletonProcessor) postRequest(msg string, releaseId string) {
	if s.UploadOutput == false {
		return
	}
	data := url.Values{}
	data.Set("action", "output")
	data.Add("release_id", releaseId)
	data.Add("msg", msg)

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
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func (s *SingletonProcessor) postOutput(releaseId string, outputChn *chan string, outputWg *sync.WaitGroup) {
	payload := ""
	postTs := time.Now()

	for {
		select {
		case msg := <-*outputChn:
			if msg == "----EOF----" {
				if payload != "" {
					s.postRequest(payload, releaseId)
				}
				close(*outputChn)
				outputWg.Done()
				return
			}

			payload = payload + msg + "\n"
			if time.Since(postTs) > time.Duration(time.Second) {
				s.postRequest(payload, releaseId)
				payload = ""
				postTs = time.Now()
			}

		case <-time.After(time.Second):
			if payload != "" {
				s.postRequest(payload, releaseId)
				payload = ""
				postTs = time.Now()
			}
		}
	}
}

func (s *SingletonProcessor) ExecuteCommand(rec *recorder.Msgbody) error {
	var err error
	var execErr error

	fname, err := createTmpFile(s.ChnName, []byte(rec.ScriptContent))
	if err != nil {
		s.Logger.Fatal("Unable creating temp file to run command: " + err.Error())
	}
	defer removeTmpFile(fname)

	command := strings.Replace(s.Command, "%releaseid%", strconv.FormatInt(rec.Id, 10), -1)
	command = strings.Replace(command, "%channel%", s.ChnName, -1)
	command = strings.Replace(command, "%meta%", rec.Meta, -1)
	command = strings.Replace(command, "%scriptpath%", fname, -1)
	command = strings.Replace(command, "%comment%", rec.Comment, -1)
	command = strings.Replace(command, "%content%", rec.ScriptContent, -1)

	for i := 1; i <= s.Retries+1; i++ {

		var scannerWg sync.WaitGroup
		var outputWg sync.WaitGroup
		outputChn := make(chan string, 10000)
		scannerWg.Add(2)
		outputWg.Add(1)

		s.Logger.Info("Executing #" + strconv.Itoa(i))

		cmd := exec.Command("sh", "-c", command)

		cmd.Dir = "/tmp"

		cmd.Env = make([]string, len(rec.Env))
		for k, v := range rec.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}

		stderr, _ := cmd.StderrPipe()
		stdout, _ := cmd.StdoutPipe()
		go func() {
			stderrScanner := bufio.NewScanner(stderr)
			for stderrScanner.Scan() {
				outputChn <- stderrScanner.Text()
			}
			scannerWg.Done()
		}()
		go func() {
			stdoutScanner := bufio.NewScanner(stdout)
			for stdoutScanner.Scan() {
				outputChn <- stdoutScanner.Text()
			}
			scannerWg.Done()
		}()

		go s.postOutput(strconv.FormatInt(rec.Id, 10), &outputChn, &outputWg)
		execErr = cmd.Start()

		if execErr != nil {
			if i < s.Retries+1 {
				s.Logger.Warning("Failed to start command, will retry after " + s.RetryDelay.String())
				time.Sleep(s.RetryDelay)
			} else {
				s.Logger.Error("All retries failed at last, aborted!")
			}
			continue
		}

		scannerWg.Wait()
		execErr = cmd.Wait()

		outputChn <- "----EOF----"
		outputWg.Wait()

		if execErr != nil {
			if i < s.Retries+1 {
				s.Logger.Warning("Failed to execute command, will retry after " + s.RetryDelay.String())
				time.Sleep(s.RetryDelay)
			} else {
				s.Logger.Error("All retries failed at last, aborted!")
			}
			continue
		} else {
			break
		}

	}

	return execErr
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
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func (s *SingletonProcessor) heartbeat() {
	for {
		s.fetchInfo()
		time.Sleep(2 * time.Second)
	}
}

func (s *SingletonProcessor) RunOnce() error {
	go s.heartbeat()
	body := <-s.MsgChann
	if len(body) == 0 {
		return nil
	}
	record, err := recorder.ParseBody(body)
	if err != nil {
		s.Logger.Error("Unrecognized Message Received(" + err.Error() + "): " + string(body))
		return err
	}

	if !s.Recorder.Equal(body) {
		s.Logger.Info("Received new command from server, creating subtask...")
		err = s.ExecuteCommand(record)
		if err != nil {
			s.Logger.Error("An error occured when running command, aborted: " + err.Error())
			return err
		}
		s.Logger.Info("Command finished and exit")
		s.Recorder.Set(body)
	}
	return nil
}

func (s *SingletonProcessor) Run() {
	go s.heartbeat()

	for body := range s.MsgChann {
		if len(body) == 0 {
			continue
		}
		record, err := recorder.ParseBody(body)
		if err != nil {
			s.Logger.Error("Unrecognized Message Received(" + err.Error() + "): " + string(body))
			continue
		}

		if !s.Recorder.Equal(body) {
			s.Logger.Info("Received new command from server, creating subtask...")
			err = s.ExecuteCommand(record)
			if err != nil {
				s.Logger.Error("An error occured when running command,aborted: " + err.Error())
			} else {
				s.Logger.Info("Command finished and exit")
				s.Recorder.Set(body)
			}
		}
	}
}
