/* Copyright (c) 2016 Chris Smith
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 *
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED ``AS IS'' AND ANY EXPRESS OR IMPLIED
 * WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY DIRECT,
 * INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
 * SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
 * STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING
 * IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package unifiedbeat

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/elastic/beats/libbeat/logp"
)

// Registrar should only have one entry, which
// is the offset into the unified2 file
// currently being tailed (if any)
type Registrar struct {
	registryFile string    // path to the registry file
	State        FileState // unified2 file name and offset
	sync.Mutex             // lock and unlock during writes
}

// remove the ",omitempty"s so something is written
// to the registry file instead of just "{}"
type FileState struct {
	Offset int64  `json:"offset"`
	Source string `json:"source"`
}

func NewRegistrar(registryFile string) (*Registrar, error) {
	r := &Registrar{
		registryFile: registryFile,
	}

	// Ensure we have access to write the registry file
	// by creating, closing, and removing a test file.
	// Of course, access could still fail in later
	// calls to  LoadState or WriteRegistry.
	// There is no perfect solution as files and
	// permissions are just a mess, but we tried.
	testfile := r.registryFile + ".access.test"
	file, err := os.Create(testfile)
	if err != nil {
		logp.Info("NewRegistrar: test 'create file' access was denied to path for registry file: '%v'\n", r.registryFile)
		return nil, err
	}
	err = file.Close()
	if err != nil {
		// really? we lost access after Create, really?
		logp.Info("NewRegistrar: test 'close file' access was denied to path for registry file: '%v'\n", r.registryFile)
		return nil, err
	}
	err = os.Remove(testfile)
	if err != nil {
		// really? we lost access after Create and Close, really?
		logp.Info("NewRegistrar: test 'remove file' access was denied to path for registry file: '%v'\n", r.registryFile)
		return nil, err
	}

	// set an absolute path to the registryFile:
	absPath, err := filepath.Abs(r.registryFile)
	if err != nil {
		logp.Info("NewRegistrar: failed to set the absolute path for registry file: '%s'\n", r.registryFile)
		return nil, err
	}
	r.registryFile = absPath

	return r, err
}

func (r *Registrar) LoadState() {
	if existing, e := os.Open(r.registryFile); e == nil {
		defer existing.Close()
		decoder := json.NewDecoder(existing)
		decoder.Decode(&r.State)
	}
}

func (r *Registrar) WriteRegistry() error {
	r.Lock()
	defer r.Unlock()
	// can't truncate a file that does not exist:
	_, err := os.Stat(r.registryFile)
	if os.IsExist(err) {
		err := os.Truncate(r.registryFile, 0)
		if err != nil {
			logp.Info("WriteRegistry: os.Truncate: err=%v\n", err)
			return err
		}
	}
	// if "json.Marshal" or "ioutil.WriteFile" fail then most likely
	// unifiedbeat does not have access to the registry file
	jsonState, err := json.Marshal(r.State)
	if err != nil {
		logp.Info("WriteRegistry: json.Marshal: err=%v\n", err)
		return err
	}
	// https://golang.org/pkg/io/ioutil/#WriteFile
	// If the file does not exist, WriteFile creates it with
	// permissions 0644; otherwise it is truncated.
	err = ioutil.WriteFile(r.registryFile, jsonState, 0644)
	if err != nil {
		logp.Info("WriteRegistry: ioutil.WriteFile: err=%v\n", err)
		return err
	}

	return nil
}
