// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Original source: github.com/micro/go-micro/v3/events/store/store.go

package store

import (
	"encoding/json"
	"time"

	"github.com/micro/micro/v3/service/events"
	gostore "github.com/micro/micro/v3/service/store"
	"github.com/micro/micro/v3/service/store/memory"
	"github.com/pkg/errors"
)

const joinKey = "/"

// NewStore returns an initialized events store
func NewStore(opts ...Option) events.Store {
	// parse the options
	var options Options
	for _, o := range opts {
		o(&options)
	}
	if options.TTL.Seconds() == 0 {
		options.TTL = time.Hour * 24
	}
	if options.Store == nil {
		options.Store = memory.NewStore()
	}

	// return the store
	return &evStore{options}
}

type evStore struct {
	opts Options
}

// Read events for a topic
func (s *evStore) Read(topic string, opts ...events.ReadOption) ([]*events.Event, error) {
	// validate the topic
	if len(topic) == 0 {
		return nil, events.ErrMissingTopic
	}

	// parse the options
	options := events.ReadOptions{
		Offset: 0,
		Limit:  250,
	}
	for _, o := range opts {
		o(&options)
	}

	// execute the request
	recs, err := s.opts.Store.Read(topic+joinKey,
		gostore.ReadPrefix(),
		gostore.ReadLimit(options.Limit),
		gostore.ReadOffset(options.Offset),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading from store")
	}

	// unmarshal the result
	result := make([]*events.Event, len(recs))
	for i, r := range recs {
		var e events.Event
		if err := json.Unmarshal(r.Value, &e); err != nil {
			return nil, errors.Wrap(err, "Invalid event returned from stroe")
		}
		result[i] = &e
	}

	return result, nil
}

// Write an event to the store
func (s *evStore) Write(event *events.Event, opts ...events.WriteOption) error {
	// parse the options
	options := events.WriteOptions{
		TTL: s.opts.TTL,
	}
	for _, o := range opts {
		o(&options)
	}

	// construct the store record
	bytes, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "Error mashaling event to JSON")
	}
	record := &gostore.Record{
		Key:    event.Topic + joinKey + event.ID,
		Value:  bytes,
		Expiry: options.TTL,
	}

	// write the record to the store
	if err := s.opts.Store.Write(record); err != nil {
		return errors.Wrap(err, "Error writing to the store")
	}

	return nil
}
