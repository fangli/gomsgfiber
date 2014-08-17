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

package recorder

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

type Record struct {
	CacheFile string
}

type Msgbody struct {
	Id            int64             `json:"id"`
	Timestamp     float64           `json:"timestamp"`
	ScriptContent string            `json:"script_content"`
	Env           map[string]string `json:"env"`
	Meta          string            `json:"meta"`
	Comment       string            `json:"comment"`
}

func (r *Record) GetRaw() []byte {
	_, err := os.Stat(r.CacheFile)
	if err != nil {
		return []byte("")
	}
	msg, err := ioutil.ReadFile(r.CacheFile)
	if err != nil {
		return []byte("")
	}
	return msg
}

func (r *Record) Set(data []byte) error {
	return ioutil.WriteFile(r.CacheFile, data, os.ModePerm)
}

func (r *Record) Equal(data []byte) bool {
	if bytes.Equal(data, r.GetRaw()) {
		return true
	}
	return false
}

func (r *Record) Get() (Msgbody, error) {
	var dat Msgbody
	var byt = r.GetRaw()
	if len(byt) == 0 {
		return dat, errors.New("Cache not set")
	}
	err := json.Unmarshal(r.GetRaw(), &dat)
	if err != nil {
		return dat, err
	}
	return dat, nil
}

func (r *Record) GetReleaseId() int64 {
	dat, err := r.Get()
	if err != nil {
		return 0
	}
	return dat.Id
}

func (r *Record) GetTimestamp() (float64, error) {
	dat, err := r.Get()
	if err != nil {
		return 0, err
	}
	return dat.Timestamp, nil
}

func (r *Record) GetScriptContent() (string, error) {
	dat, err := r.Get()
	if err != nil {
		return "", err
	}
	return dat.ScriptContent, nil
}

func (r *Record) GetEnv() (map[string]string, error) {
	dat, err := r.Get()
	if err != nil {
		return nil, err
	}
	return dat.Env, nil
}

func (r *Record) GetMeta() (string, error) {
	dat, err := r.Get()
	if err != nil {
		return "", err
	}
	return dat.Meta, nil
}

func (r *Record) GetComment() (string, error) {
	dat, err := r.Get()
	if err != nil {
		return "", err
	}
	return dat.Comment, nil
}

func ParseBody(body []byte) (*Msgbody, error) {
	var dat Msgbody
	err := json.Unmarshal(body, &dat)
	if err != nil {
		return nil, err
	}
	return &dat, nil
}
