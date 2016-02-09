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

// EventAggregator is used to aggregate records into events.
type EventAggregator struct {
	records []interface{}
}

// NewEventAggregator creates a new EventAggregator.
func NewEventAggregator() *EventAggregator {
	return &EventAggregator{}
}

// Add adds a record to the event aggregated returning an array of
// records comprising a single event if the new record is the start of
// a new event.
func (ea *EventAggregator) Add(record interface{}) []interface{} {

	var event []interface{}

	var isEventType bool

	_, isEventType = record.(*EventRecord)

	if ea.Len() == 0 && !isEventType {
		// Buffer is empty, and this is not an event record, toss it.
		return nil
	}

	// This is an event record, flush the buffer if there are any
	// records.
	if isEventType && ea.Len() > 0 {
		event = ea.Flush()
	}

	ea.records = append(ea.records, record)

	return event
}

// Len returns the number of records currently in the aggregator.
func (ea *EventAggregator) Len() int {
	return len(ea.records)
}

// Flush removes all records from the aggregator returning them as an
// array.
func (ea *EventAggregator) Flush() []interface{} {

	if ea.Len() == 0 {
		return nil
	}

	event := make([]interface{}, len(ea.records))
	copy(event, ea.records)
	ea.records = ea.records[:0]

	return event
}
