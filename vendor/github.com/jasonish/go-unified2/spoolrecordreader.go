/* Copyright (c) 2013 Jason Ish
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

package unified2

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

// SpoolRecordReader is a unified2 record reader that reads from a
// directory containing unified2 "spool" files.
//
// Unified2 spool files are files that have a common prefix and are
// suffixed by a timestamp.  This is the typical format used by Snort
// and Suricata as new unified2 files are closed and a new one is
// created when they reach a certain size.
type SpoolRecordReader struct {

	// CloseHook will be called when a file is closed.  It can be used
	// to delete or archive the file.
	CloseHook func(string)

	directory string
	prefix    string
	logger    *log.Logger
	reader    *RecordReader
}

// NewSpoolRecordReader creates a new RecordSpoolReader reading files
// prefixed with the provided prefix in the passed in directory.
func NewSpoolRecordReader(directory string, prefix string) *SpoolRecordReader {
	reader := new(SpoolRecordReader)
	reader.directory = directory
	reader.prefix = prefix
	return reader
}

func (r *SpoolRecordReader) log(format string, v ...interface{}) {
	if r.logger != nil {
		r.logger.Printf(format, v...)
	}
}

// Logger sets a logger.  Useful while testing/debugging.
func (r *SpoolRecordReader) Logger(logger *log.Logger) {
	r.logger = logger
}

// getFiles returns a sorted list of filename in the spool
// directory with the specified prefix.
func (r *SpoolRecordReader) getFiles() ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(r.directory)
	if err != nil {
		return nil, err
	}

	filtered := make([]os.FileInfo, len(files))
	filtered_idx := 0

	for _, file := range files {
		if strings.HasPrefix(file.Name(), r.prefix) {
			filtered[filtered_idx] = file
			filtered_idx++
		}
	}

	return filtered[0:filtered_idx], nil
}

// openNext opens the next available file if it exists.  If a new file
// is opened its filename will be returned.
func (r *SpoolRecordReader) openNext() bool {
	files, err := r.getFiles()
	if err != nil {
		r.log("Failed to get filenames: %s", err)
		return false
	}

	if len(files) == 0 {
		// Nothing to do.
		return false
	}

	if r.reader != nil {
		r.log("Currently open file: %s", r.reader.Name())
	}

	var nextFilename string

	for _, file := range files {
		if r.reader == nil {
			nextFilename = path.Join(r.directory, file.Name())
			break
		} else {
			if path.Base(r.reader.Name()) != file.Name() {
				nextFilename = path.Join(r.directory, file.Name())
				break
			}
		}
	}

	if nextFilename == "" {
		r.log("No new files found.")
		return false
	}

	if r.reader != nil {
		r.log("Closing %s.", r.reader.Name())
		r.reader.Close()

		// Call the close hook if set.
		if r.CloseHook != nil {
			r.CloseHook(r.reader.Name())
		}
	}

	r.log("Opening file %s", nextFilename)
	r.reader, err = NewRecordReader(nextFilename, 0)
	if err != nil {
		r.log("Failed to open %s: %s", nextFilename, err)
		return false
	}
	return true
}

// Next returns the next record read from the spool.
func (r *SpoolRecordReader) Next() (interface{}, error) {

	for {

		// If we have no current file, try to open one.
		if r.reader == nil {
			r.openNext()
		}

		// If we still don't have a current file, return.
		if r.reader == nil {
			return nil, nil
		}

		record, err := r.reader.Next()

		if err == io.EOF {
			if r.openNext() {
				continue
			}
		}

		return record, err

	}

}

// Offset returns the current filename that is being processed and its
// read position (the offset).
func (r *SpoolRecordReader) Offset() (string, int64) {
	if r.reader != nil {
		return path.Base(r.reader.Name()), r.reader.Offset()
	} else {
		return "", 0
	}
}
