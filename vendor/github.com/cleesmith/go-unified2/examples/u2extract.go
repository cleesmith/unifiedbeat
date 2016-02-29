// Extract events from a unified2 log file with the specified event-id
// and event-second.
package main

import "os"
import "flag"
import "log"
import "io"
import "github.com/jasonish/go-unified2"
import "encoding/binary"

func writeRecord(out *os.File, record *unified2.RawRecord) (err error) {

	recordType := record.Type
	recordLen := uint32(len(record.Data))

	err = binary.Write(out, binary.BigEndian, &recordType)
	if err != nil {
		return err
	}

	err = binary.Write(out, binary.BigEndian, &recordLen)
	if err != nil {
		return err
	}

	n, err := out.Write(record.Data)
	if err != nil {
		return err
	} else if n != len(record.Data) {
		return io.ErrShortWrite
	}

	return nil
}

func main() {

	var filterEventId uint
	var eventSecond uint

	flag.UintVar(&filterEventId, "event-id", 0, "filter on event-id")
	flag.UintVar(&eventSecond, "event-second", 0, "filter on event-secon")
	flag.Parse()

	if filterEventId == 0 || eventSecond == 0 {
		log.Fatalf("error: both -event-id and -event-second must be specified")
	}

	args := flag.Args()

	var written uint

	for _, arg := range args {

		file, err := os.Open(arg)
		if err != nil {
			log.Fatal(err)
		}

		var currentEvent *unified2.EventRecord

		for {
			/* Want to read the raw record and decode it separately,
			/* so we can write out the raw records. */

			raw, err := unified2.ReadRawRecord(file)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatal(err)
			}

			switch raw.Type {
			case unified2.UNIFIED2_IDS_EVENT,
				unified2.UNIFIED2_IDS_EVENT_IP6,
				unified2.UNIFIED2_IDS_EVENT_V2,
				unified2.UNIFIED2_IDS_EVENT_IP6_V2:
				event, err := unified2.DecodeEventRecord(raw.Type, raw.Data)
				if err != nil {
					log.Fatalf("failed to decode event")
				}

				/* Filter. */
				if uint32(filterEventId) == event.EventId &&
					uint32(eventSecond) == event.EventSecond {
					currentEvent = event
				} else {
					currentEvent = nil
				}
			}

			if currentEvent != nil {
				writeRecord(os.Stdout, raw)
				written++
			}

		}

		file.Close()

	}

	log.Printf("Records written: %d\n", written)

}
