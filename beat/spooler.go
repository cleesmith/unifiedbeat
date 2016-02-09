package beat

import (
	"time"

	"github.com/elastic/libbeat/logp"
	cfg "github.com/elastic/unifiedbeat/config"
	"github.com/elastic/unifiedbeat/input"
)

type Spooler struct {
	Unifiedbeat   *Unifiedbeat
	running       bool
	nextFlushTime time.Time
	spool         []*input.FileEvent
	Channel       chan *input.FileEvent
}

func NewSpooler(unifiedbeat *Unifiedbeat) *Spooler {
	spooler := &Spooler{
		Unifiedbeat: unifiedbeat,
		running:     false,
	}

	config := &spooler.Unifiedbeat.UbConfig.Unifiedbeat

	// Set the next flush time
	spooler.nextFlushTime = time.Now().Add(config.IdleTimeoutDuration)
	spooler.Channel = make(chan *input.FileEvent, 16)

	return spooler
}

func (spooler *Spooler) Config() error {
	config := &spooler.Unifiedbeat.UbConfig.Unifiedbeat

	// Set default pool size if value not set
	if config.SpoolSize == 0 {
		config.SpoolSize = cfg.DefaultSpoolSize
	}

	// Set default idle timeout if not set
	if config.IdleTimeout == "" {
		logp.Debug("spooler", "Set idleTimeoutDuration to %s", cfg.DefaultIdleTimeout)
		// Set it to default
		config.IdleTimeoutDuration = cfg.DefaultIdleTimeout
	} else {
		var err error

		config.IdleTimeoutDuration, err = time.ParseDuration(config.IdleTimeout)

		if err != nil {
			logp.Warn("Failed to parse idle timeout duration '%s'. Error was: %v", config.IdleTimeout, err)
			return err
		}
	}

	return nil
}

// Run runs the spooler
// It heartbeats periodically. If the last flush was longer than
// 'IdleTimeoutDuration' time ago, then we'll force a flush to prevent us from
// holding on to spooled events for too long.
func (s *Spooler) Run() {
	config := &s.Unifiedbeat.UbConfig.Unifiedbeat

	// Enable running
	s.running = true

	// Sets up ticker channel
	ticker := time.NewTicker(config.IdleTimeoutDuration / 2)

	s.spool = make([]*input.FileEvent, 0, config.SpoolSize)

	logp.Info("Starting spooler: spool_size: %v; idle_timeout: %s", config.SpoolSize, config.IdleTimeoutDuration)

	// Loops until running is set to false
	for {
		if !s.running {
			break
		}

		select {
		case event := <-s.Channel:
			s.spool = append(s.spool, event)
			// Spooler is full -> flush
			if len(s.spool) == cap(s.spool) {
				logp.Debug("spooler", "Flushing spooler because spooler full. Events flushed: %v", len(s.spool))
				s.flush()
			}
		case <-ticker.C:
			// Flush periodically
			if time.Now().After(s.nextFlushTime) {
				logp.Debug("spooler", "Flushing spooler because of timemout. Events flushed: %v", len(s.spool))
				s.flush()
			}
		}
	}

	logp.Info("Stopping spooler")

	// Flush again before exiting spooler and closes channel
	s.flush()
	close(s.Channel)
}

// Stop stops the spooler. Flushes events before stopping
func (s *Spooler) Stop() {
}

// flush flushes all event and sends them to the publisher
func (s *Spooler) flush() {
	// Checks if any new objects
	if len(s.spool) > 0 {

		// copy buffer
		tmpCopy := make([]*input.FileEvent, len(s.spool))
		copy(tmpCopy, s.spool)

		// clear buffer
		s.spool = s.spool[:0]

		// send
		s.Unifiedbeat.publisherChan <- tmpCopy
	}
	s.nextFlushTime = time.Now().Add(s.Unifiedbeat.UbConfig.Unifiedbeat.IdleTimeoutDuration)
}
