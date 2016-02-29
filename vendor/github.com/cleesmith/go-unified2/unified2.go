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

/*

Package unified2 provides a decoder for unified v2 log files
produced by Snort and Suricata.

Currently the unified2 package provides:
    - Unified2 record decoders.
    - A record aggregator to aggregate records into an event.

*/
package unified2

import (
	"encoding/binary"
	"io"
)

// Unified2 record types.
const (
	UNIFIED2_PACKET           = 2
	UNIFIED2_IDS_EVENT        = 7
	UNIFIED2_IDS_EVENT_IP6    = 72
	UNIFIED2_IDS_EVENT_V2     = 104
	UNIFIED2_IDS_EVENT_IP6_V2 = 105
	UNIFIED2_EXTRA_DATA       = 110
)

// RawHeader is the raw unified2 record header.
type RawHeader struct {
	Type uint32
	Len  uint32
}

// RawRecord is a holder type for a raw un-decoded record.
type RawRecord struct {
	Type uint32
	Data []byte
}

// EventRecord is a struct representing a decoded event record.
//
// This struct is used to represent the decoded form of all the event
// types.  The difference between an IPv4 and IPv6 event will be the
// length of the IP address IpSource and IpDestination.
type EventRecord struct {
	SensorId          uint32
	EventId           uint32
	EventSecond       uint32
	EventMicrosecond  uint32
	SignatureId       uint32
	GeneratorId       uint32
	SignatureRevision uint32
	ClassificationId  uint32
	Priority          uint32
	IpSource          []byte
	IpDestination     []byte
	SportItype        uint16
	DportIcode        uint16
	Protocol          uint8
	ImpactFlag        uint8
	Impact            uint8
	Blocked           uint8
	MplsLabel         uint32
	VlanId            uint16
}

// PacketRecord is a struct representing a decoded packet record.
type PacketRecord struct {
	SensorId          uint32
	EventId           uint32
	EventSecond       uint32
	PacketSecond      uint32
	PacketMicrosecond uint32
	LinkType          uint32
	Length            uint32
	Data              []byte
}

// The length of a PacketRecord before variable length data.
const PACKET_RECORD_HDR_LEN = 28

// ExtraDataRecord is a struct representing a decoded extra data record.
type ExtraDataRecord struct {
	EventType   uint32
	EventLength uint32
	SensorId    uint32
	EventId     uint32
	EventSecond uint32
	Type        uint32
	DataType    uint32
	DataLength  uint32
	Data        []byte
}

// The length of an ExtraDataRecord before variable length data.
const EXTRA_DATA_RECORD_HDR_LEN = 32

// ReadRawRecord reads a raw record from the provided file.
//
// On error, err will no non-nil.  Expected error values are io.EOF
// when the end of the file has been reached or io.ErrUnexpectedEOF if
// a complete record was unable to be read.
//
// In the case of io.ErrUnexpectedEOF the file offset will be reset
// back to where it was upon entering this function so it is ready to
// be read from again if it is expected more data will be written to
// the file.
func ReadRawRecord(file io.ReadWriteSeeker) (*RawRecord, error) {
	var header RawHeader

	/* Get the current offset so we can seek back to it. */
	offset, _ := file.Seek(0, 1)

	/* Now read in the header. */
	err := binary.Read(file, binary.BigEndian, &header)
	if err != nil {
		file.Seek(offset, 0)
		return nil, err
	}

	/* Create a buffer to hold the raw record data and read the
	/* record data into it */
	data := make([]byte, header.Len)
	n, err := file.Read(data)
	if err != nil {
		file.Seek(offset, 0)
		return nil, err
	}
	if uint32(n) != header.Len {
		file.Seek(offset, 0)
		return nil, io.ErrUnexpectedEOF
	}

	return &RawRecord{header.Type, data}, nil
}

// ReadRecord reads a record from the provided file and returns a
// decoded record.
//
// On error, err will no non-nil.  Expected error values are io.EOF
// when the end of the file has been reached or io.ErrUnexpectedEOF if
// a complete record was unable to be read.
//
// In the case of io.ErrUnexpectedEOF the file offset will be reset
// back to where it was upon entering this function so it is ready to
// be read from again if it is expected more data will be written to
// the file.
//
// If an error occurred during decoding of the read data a
// DecodingError will be returned.  This likely means the input is
// corrupt.
func ReadRecord(file io.ReadWriteSeeker) (interface{}, error) {

	record, err := ReadRawRecord(file)
	if err != nil {
		return nil, err
	}

	var decoded interface{}

	switch record.Type {
	case UNIFIED2_IDS_EVENT,
		UNIFIED2_IDS_EVENT_IP6,
		UNIFIED2_IDS_EVENT_V2,
		UNIFIED2_IDS_EVENT_IP6_V2:
		decoded, err = DecodeEventRecord(record.Type, record.Data)
	case UNIFIED2_PACKET:
		decoded, err = DecodePacketRecord(record.Data)
	case UNIFIED2_EXTRA_DATA:
		decoded, err = DecodeExtraDataRecord(record.Data)
	}

	if err != nil {
		return nil, err
	} else if decoded != nil {
		return decoded, nil
	} else {
		// Unknown record type.
		return nil, nil
	}
}
