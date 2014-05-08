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
	"io/ioutil"
	"os"
)

type Record struct {
	CachePath string
}

func (r *Record) Get(name string) []byte {
	var fname = r.CachePath + "/" + name
	_, err := os.Stat(fname)
	if err != nil {
		return []byte("")
	}
	msg, err := ioutil.ReadFile(fname)
	if err != nil {
		return []byte("")
	}
	return msg
}

func (r *Record) Set(name string, data []byte) error {
	var fname = r.CachePath + "/" + name
	return ioutil.WriteFile(fname, data, os.ModePerm)
}

func (r *Record) Equal(name string, data []byte) bool {
	if bytes.Equal(data, r.Get(name)) {
		return true
	}
	return false
}
