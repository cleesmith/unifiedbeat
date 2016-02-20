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
	"fmt"
	"io"
	// "log"
	"os"
	"path"
	"time"

	"github.com/cleesmith/go-unified2"

	"github.com/elastic/beats/libbeat/logp"
)

// "Spool" refers to handling a folder of unified2 files
// in ascending order by filename as a continous
// stream of records to be read and indexed.
// Well, that's not whole story, as it is aware of each
// file being indexed and will call the CloseHook func
// when one is provided. CloseHook allows the program
// to "archive/rename" the indexed file and timestamp it,
// which avoids continuously looping over the same data
// leading to document duplication.
func (ub *Unifiedbeat) U2SpoolAndPublish() {
	logp.Info("U2SpoolAndPublish: spooling and publishing...")
	reader := unified2.NewSpoolRecordReader(ub.UbConfig.Sensor.Spooler.Folder,
		ub.UbConfig.Sensor.Spooler.FilePrefix)
	// only for debugging:
	// reader.Logger(log.New(os.Stdout, "SpoolRecordReader: ", 0))

	closeHookCount := 0
	reader.CloseHook = func(filepath string) {
		closeHookCount++
		filedir := path.Dir(filepath)
		filename := path.Base(filepath)
		newname := fmt.Sprintf("/indexed_%v.", time.Now().Unix())
		filepathRename := filedir + newname + filename
		err := os.Rename(filepath, filepathRename)
		if err != nil {
			logp.Info("unable to rename file '%v' to '%v' err: %v", filepath, filepathRename, err)
			return
		}
		logp.Info("Indexed file: '%v' renamed: '%v'", filepath, filepathRename)
	}

	// use current registrar state:
	reader.FileSource = ub.registrar.State.Source
	reader.FileOffset = ub.registrar.State.Offset

	var tot int
	// forever index all files in the specifed spool folder:
	for ub.isSpooling {
		// select {
		// case <-quit:
		// 	logp.Info("U2SpoolAndPublish: told to quit; graceful return.")
		// 	close(quit)
		// 	return
		// default:
		record, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				// EOF is returned when the end of the last (most recent file)
				// spool file is reached and there is nothing else to read.
				// Note that "reader.Next()" only returns "io.EOF" when there
				// are no other files to open ... in other words, it is
				// always tailing the last file opened.
				time.Sleep(time.Millisecond * 500)
			} else {
				logp.Critical("U2SpoolAndPublish: unexpected error: '%v'", err)
				return
			}
		}

		if record == nil {
			// The vars "record" and "err" are nil when there are no files
			// at all to be read.  This will happen if the "reader.Next()"
			// is called before any files exist in the folder being spooled.
			time.Sleep(time.Millisecond * 500)
			// now, go see if a new record has appeared
			continue
		}

		// at this point, we have read a unified2 record, which
		// needs to be converted into JSON and indexed into ES
		filename, offset := reader.Offset()

		// update registrar:
		ub.registrar.State.Source = filename
		ub.registrar.State.Offset = offset
		// should it WriteRegistry here ?
		// that means lots of disk writes, but is
		// the registry file info that important ?

		tot++
		sourceFullPath := path.Join(ub.UbConfig.Sensor.Spooler.Folder, filename)
		event := &FileEvent{
			ReadTime:     time.Now(),
			Source:       sourceFullPath,
			InputType:    "unified2",
			DocumentType: "unified2", // this changes for each unified2 record type
			Offset:       offset,
			U2Record:     record,
			Fields:       &ub.UbConfig.Sensor.Fields,
		}
		event.SetFieldsUnderRoot(ub.UbConfig.Sensor.FieldsUnderRoot)

		eventCommonMapStr := event.ToMapStr() // see "beat/u2recordhandler.go"

		ub.events.PublishEvent(eventCommonMapStr)

		// } // end: select default

	} // end: forever index all files in the specifed spool folder

	logp.Info("U2SpoolAndPublish: done.")
}
