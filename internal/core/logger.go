package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/shniranjan/lightboot/internal/event"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// LogEntry is a single log message stored in the ring buffer.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}

// LogRingBuffer is a thread-safe fixed-size ring buffer for log entries.
type LogRingBuffer struct {
	mu       sync.RWMutex
	entries  []LogEntry
	maxSize  int
	writeIdx int
	full     bool
}

// NewLogRingBuffer creates a new ring buffer with the given maximum size.
func NewLogRingBuffer(maxSize int) *LogRingBuffer {
	return &LogRingBuffer{
		entries: make([]LogEntry, maxSize),
		maxSize: maxSize,
	}
}

// Push adds a log entry to the buffer. If the buffer is full, the oldest
// entry is overwritten.
func (b *LogRingBuffer) Push(entry LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry.Timestamp = time.Now()
	b.entries[b.writeIdx] = entry
	b.writeIdx = (b.writeIdx + 1) % b.maxSize
	if b.writeIdx == 0 {
		b.full = true
	}
}

// Snapshot returns all log entries in chronological order.
func (b *LogRingBuffer) Snapshot() []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.full {
		result := make([]LogEntry, b.writeIdx)
		copy(result, b.entries[:b.writeIdx])
		return result
	}

	result := make([]LogEntry, b.maxSize)
	copy(result, b.entries[b.writeIdx:])
	copy(result[b.maxSize-b.writeIdx:], b.entries[:b.writeIdx])
	return result
}

// Recent returns up to n most recent entries.
func (b *LogRingBuffer) Recent(n int) []LogEntry {
	all := b.Snapshot()
	if len(all) <= n {
		return all
	}
	return all[len(all)-n:]
}

// Logger sends log entries both to the ring buffer and to the event bus
// and optionally a file so SSE subscribers and disk logs work together.
type Logger struct {
	buffer *LogRingBuffer
	bus    *event.EventBus
	level  LogLevel
	file   *os.File
	mu     sync.Mutex
}

// NewLogger creates a new Logger.
func NewLogger(buffer *LogRingBuffer, bus *event.EventBus, level LogLevel) *Logger {
	return &Logger{
		buffer: buffer,
		bus:    bus,
		level:  level,
	}
}

// SetLogFile opens a file for writing log entries. Call Close() on shutdown.
func (l *Logger) SetLogFile(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", path, err)
	}
	l.file = f
	return nil
}

// Close closes the log file if one was opened.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) log(level LogLevel, source, format string, args ...any) {
	entry := LogEntry{
		Level:   level,
		Source:  source,
		Message: fmt.Sprintf(format, args...),
	}

	l.buffer.Push(entry)
	l.bus.Publish(event.LogEntry, entry)

	// Write to file if configured
	l.mu.Lock()
	if l.file != nil {
		line := fmt.Sprintf("[%s] [%s] [%s] %s\n",
			time.Now().Format(time.RFC3339),
			level,
			source,
			entry.Message)
		l.file.WriteString(line)
	}
	l.mu.Unlock()
}

func (l *Logger) Debug(source, format string, args ...any) {
	if l.level == LogDebug {
		l.log(LogDebug, source, format, args...)
	}
}

func (l *Logger) Info(source, format string, args ...any) {
	l.log(LogInfo, source, format, args...)
}

func (l *Logger) Warn(source, format string, args ...any) {
	l.log(LogWarn, source, format, args...)
}

func (l *Logger) Error(source, format string, args ...any) {
	l.log(LogError, source, format, args...)
}

// SSEHandler returns an http.HandlerFunc that streams log entries via SSE.
type SSEHandler struct {
	bus    *event.EventBus
	buffer *LogRingBuffer
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(bus *event.EventBus, buffer *LogRingBuffer) *SSEHandler {
	return &SSEHandler{bus: bus, buffer: buffer}
}

// ServeHTTP implements http.Handler for the SSE endpoint.
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send recent entries first
	for _, entry := range h.buffer.Recent(20) {
		data, _ := json.Marshal(entry)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}
	flusher.Flush()

	// Subscribe to new log entries
	sub := h.bus.Subscribe(event.LogEntry)
	defer h.bus.Unsubscribe(event.LogEntry, sub)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sub:
			if !ok {
				return
			}
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
