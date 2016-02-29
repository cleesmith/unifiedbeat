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
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// DecodingError is the error returned if an error is encountered
// while decoding a record buffer.
//
// We use this error to differentiate between file level reading
// errors.
var DecodingError = errors.New("DecodingError")

// Helper function for reading binary data as all reads are big
// endian.
func read(reader io.Reader, data interface{}) error {
	return binary.Read(reader, binary.BigEndian, data)
}

// DecodeEventRecord decodes a raw record into an EventRecord.
//
// This function will decode any of the event record types.
func DecodeEventRecord(eventType uint32, data []byte) (*EventRecord, error) {

	event := &EventRecord{}

	reader := bytes.NewBuffer(data)

	// SensorId
	if err := read(reader, &event.SensorId); err != nil {
		goto error
	}
	if err := read(reader, &event.EventId); err != nil {
		goto error
	}
	if err := read(reader, &event.EventSecond); err != nil {
		goto error
	}
	if err := read(reader, &event.EventMicrosecond); err != nil {
		goto error
	}

	/* SignatureId */
	if err := read(reader, &event.SignatureId); err != nil {
		goto error
	}

	/* GeneratorId */
	if err := read(reader, &event.GeneratorId); err != nil {
		goto error
	}

	/* SignatureRevision */
	if err := read(reader, &event.SignatureRevision); err != nil {
		goto error
	}

	/* ClassificationId */
	if err := read(reader, &event.ClassificationId); err != nil {
		goto error
	}

	/* Priority */
	if err := read(reader, &event.Priority); err != nil {
		goto error
	}

	/* Source and destination IP addresses. */
	switch eventType {

	case UNIFIED2_IDS_EVENT, UNIFIED2_IDS_EVENT_V2:
		event.IpSource = make([]byte, 4)
		if err := read(reader, &event.IpSource); err != nil {
			goto error
		}
		event.IpDestination = make([]byte, 4)
		if err := read(reader, &event.IpDestination); err != nil {
			goto error
		}

	case UNIFIED2_IDS_EVENT_IP6, UNIFIED2_IDS_EVENT_IP6_V2:
		event.IpSource = make([]byte, 16)
		if err := read(reader, &event.IpSource); err != nil {
			goto error
		}
		event.IpDestination = make([]byte, 16)
		if err := read(reader, &event.IpDestination); err != nil {
			goto error
		}
	}

	/* Source port/ICMP type. */
	if err := read(reader, &event.SportItype); err != nil {
		goto error
	}

	/* Destination port/ICMP code. */
	if err := read(reader, &event.DportIcode); err != nil {
		goto error
	}

	/* Protocol. */
	if err := read(reader, &event.Protocol); err != nil {
		goto error
	}

	/* Impact flag. */
	if err := read(reader, &event.ImpactFlag); err != nil {
		goto error
	}

	/* Impact. */
	if err := read(reader, &event.Impact); err != nil {
		goto error
	}

	/* Blocked. */
	if err := read(reader, &event.Blocked); err != nil {
		goto error
	}

	switch eventType {
	case UNIFIED2_IDS_EVENT_V2, UNIFIED2_IDS_EVENT_IP6_V2:

		/* MplsLabel. */
		if err := read(reader, &event.MplsLabel); err != nil {
			goto error
		}

		/* VlanId. */
		if err := read(reader, &event.VlanId); err != nil {
			goto error
		}

	}

	return event, nil

error:
	return nil, DecodingError
}

// DecodePacketRecord decodes a raw unified2 record into a
// PacketRecord.
func DecodePacketRecord(data []byte) (packet *PacketRecord, err error) {

	packet = &PacketRecord{}

	reader := bytes.NewBuffer(data)

	if err = read(reader, &packet.SensorId); err != nil {
		goto error
	}

	if err = read(reader, &packet.EventId); err != nil {
		goto error
	}

	if err = read(reader, &packet.EventSecond); err != nil {
		goto error
	}

	if err = read(reader, &packet.PacketSecond); err != nil {
		goto error
	}

	if err = read(reader, &packet.PacketMicrosecond); err != nil {
		goto error
	}

	if err = read(reader, &packet.LinkType); err != nil {
		goto error
	}

	if err = read(reader, &packet.Length); err != nil {
		goto error
	}

	packet.Data = data[PACKET_RECORD_HDR_LEN:]

	return packet, nil

error:
	return nil, DecodingError
}

// DecodeExtraDataRecord decodes a raw extra data record into an
// ExtraDataRecord.
func DecodeExtraDataRecord(data []byte) (extra *ExtraDataRecord, err error) {

	extra = &ExtraDataRecord{}

	reader := bytes.NewBuffer(data)

	if err = read(reader, &extra.EventType); err != nil {
		goto error
	}

	if err = read(reader, &extra.EventLength); err != nil {
		goto error
	}

	if err = read(reader, &extra.SensorId); err != nil {
		goto error
	}

	if err = read(reader, &extra.EventId); err != nil {
		goto error
	}

	if err = read(reader, &extra.EventSecond); err != nil {
		goto error
	}

	if err = read(reader, &extra.Type); err != nil {
		goto error
	}

	if err = read(reader, &extra.DataType); err != nil {
		goto error
	}

	if err = read(reader, &extra.DataLength); err != nil {
		goto error
	}

	extra.Data = data[EXTRA_DATA_RECORD_HDR_LEN:]

	return extra, nil

error:
	return nil, DecodingError
}
