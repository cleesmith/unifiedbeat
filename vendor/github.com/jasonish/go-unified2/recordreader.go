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
	"log"
	"os"
)

// RecordReader reads and decodes unified2 records from a file.
//
// RecordReaders should be created with NewRecordReader().
type RecordReader struct {
	File *os.File
}

// NewRecordReader creates a new RecordReader using the provided
// filename and starting at the provided offset.
func NewRecordReader(filename string, offset int64) (*RecordReader, error) {

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		ret, err := file.Seek(offset, 0)
		if err != nil {
			log.Printf("Failed to seek to offset %d: %s", offset, err)
			file.Close()
			return nil, err
		} else if ret != offset {
			log.Printf("Failed to seek to offset %d: current offset: %s",
				offset, ret)
			file.Close()
			return nil, err
		}
	}

	return &RecordReader{file}, nil
}

// Next reads and returns the next unified2 record.  The record is
// returned as an interface{} which will be one of the types
// EventRecord, PacketRecord or ExtraDataRecord.
func (r *RecordReader) Next() (interface{}, error) {
	return ReadRecord(r.File)
}

// Close closes this reader and the underlying file.
func (r *RecordReader) Close() {
	r.File.Close()
}

// Offset returns the current offset of this reader.
func (r *RecordReader) Offset() int64 {
	offset, err := r.File.Seek(0, 1)
	if err != nil {
		return 0
	}
	return offset
}

// Name returns the name of the file being read.
func (r *RecordReader) Name() string {
	return r.File.Name()
}
