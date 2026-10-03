package har

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// EntryIterator provides an interface for streaming through entries in a HAR file.
type EntryIterator interface {
	// Next moves to the next entry and returns false if there are no more entries.
	Next() bool
	// Entry returns the current entry.
	Entry() *Entries
	// Err returns any error encountered during iteration.
	Err() error
	// Close closes the iterator and associated resources.
	Close() error
}

// StreamingHar represents a HAR file processed as a stream.
type StreamingHar struct {
	file       *os.File
	fileOffset int64
	mutex      sync.Mutex
	creator    Creator
	browser    Browser
	pages      []Pages
	version    string
	data       []byte
}

// StreamingEntryIterator iterates over HAR entries.
type StreamingEntryIterator struct {
	har            *StreamingHar
	decoder        *json.Decoder
	err            error
	file           *os.File
	currentPos     int
	entry          Entries
	closed         bool
	entriesStarted bool
}

// NewStreamingHarFromFile creates a streaming HAR object from a file path.
func NewStreamingHarFromFile(filePath string) (*StreamingHar, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, NewFileSystemError(fmt.Sprintf("failed to open HAR file '%s'", filePath), err)
	}

	decoder := json.NewDecoder(file)
	har := &StreamingHar{file: file}

	if err := findHarObjectStart(decoder); err != nil {
		return nil, closeStreamingFileAfterError(file, NewJSONParseError("failed to parse HAR stream", err))
	}

	if err := parseHarBasicInfo(decoder, har); err != nil {
		return nil, closeStreamingFileAfterError(file, NewJSONParseError("failed to parse HAR stream metadata", err))
	}

	har.fileOffset = int64(decoder.InputOffset())
	return har, nil
}

// NewStreamingHarFromBytes creates a streaming HAR object from bytes.
func NewStreamingHarFromBytes(data []byte) (*StreamingHar, error) {
	tempHar := &Har{}
	err := json.Unmarshal(data, tempHar)
	if err != nil {
		return nil, WrapJSONUnmarshalError(err)
	}

	har := &StreamingHar{
		data:    data,
		version: tempHar.Log.Version,
		creator: tempHar.Log.Creator,
		browser: tempHar.Log.Browser,
		pages:   tempHar.Log.Pages,
	}

	return har, nil
}

func closeStreamingFileAfterError(file *os.File, harErr *HarError) *HarError {
	if closeErr := file.Close(); closeErr != nil {
		harErr = harErr.WithMetadata("closeError", closeErr.Error())
	}
	return harErr
}

func findHarObjectStart(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return errors.New("expected { at the start of HAR file")
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("failed to find log field: %w", err)
		}
		if str, ok := token.(string); ok && str == "log" {
			break
		}
	}

	token, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read token after log: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return errors.New("expected { after log field")
	}

	return nil
}

func parseHarBasicInfo(decoder *json.Decoder, har *StreamingHar) error {
	for {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("failed to read field name: %w", err)
		}

		if delim, ok := token.(json.Delim); ok && delim == '}' {
			break
		}

		// json.Decoder.Token() only returns a non-string token at an object
		// key position by reporting a syntax error (handled above), so the
		// field name is always a string here.
		fieldName := token.(string)

		switch fieldName {
		case "version":
			if err := decoder.Decode(&har.version); err != nil {
				return fmt.Errorf("failed to decode version: %w", err)
			}
		case "creator":
			if err := decoder.Decode(&har.creator); err != nil {
				return fmt.Errorf("failed to decode creator: %w", err)
			}
		case "browser":
			if err := decoder.Decode(&har.browser); err != nil {
				return fmt.Errorf("failed to decode browser: %w", err)
			}
		case "pages":
			if err := decoder.Decode(&har.pages); err != nil {
				return fmt.Errorf("failed to decode pages: %w", err)
			}
		case "entries":
			token, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("failed to find entries array start: %w", err)
			}
			if delim, ok := token.(json.Delim); !ok || delim != '[' {
				return errors.New("expected [ at the start of entries")
			}
			return nil
		default:
			var dummy interface{}
			if err := decoder.Decode(&dummy); err != nil {
				return fmt.Errorf("failed to skip field %s: %w", fieldName, err)
			}
		}
	}
	return nil
}

// Close closes the StreamingHar and releases its resources.
func (h *StreamingHar) Close() error {
	if h == nil {
		return nil
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.file != nil {
		err := h.file.Close()
		h.file = nil
		if err != nil {
			return NewFileSystemError("failed to close streaming HAR file", err)
		}
	}
	return nil
}

// GetVersion returns the HAR version.
func (h *StreamingHar) GetVersion() string {
	if h == nil {
		return ""
	}
	return h.version
}

// GetCreator returns HAR creator information.
func (h *StreamingHar) GetCreator() Creator {
	if h == nil {
		return Creator{}
	}
	return h.creator
}

// GetBrowser returns browser information.
func (h *StreamingHar) GetBrowser() Browser {
	if h == nil {
		return Browser{}
	}
	return h.browser
}

// GetPages returns page information.
func (h *StreamingHar) GetPages() []Pages {
	if h == nil {
		return nil
	}
	return h.pages
}

// Entries returns an entry iterator.
func (h *StreamingHar) Entries() *StreamingEntryIterator {
	if h == nil {
		return &StreamingEntryIterator{
			err:    NewInvalidFormatError("StreamingHar对象为空"),
			entry:  Entries{},
			closed: true,
		}
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.data != nil {
		decoder := json.NewDecoder(bytes.NewReader(h.data))
		return &StreamingEntryIterator{
			har:     h,
			decoder: decoder,
			entry:   Entries{},
		}
	}

	if h.file != nil {
		filePath := h.file.Name()
		reopenedFile, err := os.Open(filePath)
		if err != nil {
			return &StreamingEntryIterator{
				har:    h,
				err:    NewFileSystemError("failed to re-open HAR file for iteration", err),
				entry:  Entries{},
				closed: true,
			}
		}

		reopenedDecoder := json.NewDecoder(reopenedFile)

		if err := findHarObjectStart(reopenedDecoder); err != nil {
			reopenedFile.Close()
			return &StreamingEntryIterator{
				har:    h,
				file:   reopenedFile,
				err:    NewJSONParseError("failed to find HAR object start on re-open", err),
				entry:  Entries{},
				closed: true,
			}
		}

		throwawayHar := &StreamingHar{}
		if err := parseHarBasicInfo(reopenedDecoder, throwawayHar); err != nil {
			reopenedFile.Close()
			return &StreamingEntryIterator{
				har:    h,
				file:   reopenedFile,
				err:    NewJSONParseError("failed to parse to entries on re-open", err),
				entry:  Entries{},
				closed: true,
			}
		}

		return &StreamingEntryIterator{
			har:            h,
			file:           reopenedFile,
			decoder:        reopenedDecoder,
			entry:          Entries{},
			entriesStarted: true,
		}
	}

	return &StreamingEntryIterator{
		har:    h,
		err:    NewInvalidFormatError("no data source available for streaming iteration"),
		entry:  Entries{},
		closed: true,
	}
}

// Next returns the next entry.
func (it *StreamingEntryIterator) Next() bool {
	if it == nil || it.decoder == nil {
		return false
	}
	if it.closed || it.err != nil {
		return false
	}

	if !it.entriesStarted {
		found := false
		for !found {
			token, err := it.decoder.Token()
			if err != nil {
				it.err = wrapStreamingIteratorError("failed to read streaming token", err)
				return false
			}

			if str, ok := token.(string); ok && str == "entries" {
				token, err = it.decoder.Token()
				if err != nil {
					it.err = wrapStreamingIteratorError("failed to read entries array start", err)
					return false
				}
				if delim, ok := token.(json.Delim); ok && delim == '[' {
					found = true
					it.entriesStarted = true
				} else {
					it.err = NewInvalidFormatError(fmt.Sprintf("预期在'entries'字段后找到'['，但实际为: %v", token))
					return false
				}
			}
		}
	}

	if !it.decoder.More() {
		return false
	}

	var entry Entries
	if err := it.decoder.Decode(&entry); err != nil {
		it.err = wrapStreamingIteratorError("failed to decode streaming entry", err)
		return false
	}

	it.entry = entry
	it.currentPos++
	return true
}

// Entry returns the current entry.
func (it *StreamingEntryIterator) Entry() *Entries {
	if it == nil {
		return nil
	}
	return &it.entry
}

// Position returns the current position.
func (it *StreamingEntryIterator) Position() int {
	if it == nil {
		return 0
	}
	return it.currentPos
}

// Err returns any error encountered while iterating.
func (it *StreamingEntryIterator) Err() error {
	if it == nil {
		return NewInvalidFormatError("StreamingEntryIterator对象为空")
	}
	if it.err == io.EOF {
		return nil
	}
	return it.err
}

func wrapStreamingIteratorError(message string, err error) error {
	if err == nil || err == io.EOF {
		return err
	}
	return NewJSONParseError(message, err)
}

// Close closes the iterator and associated resources.
func (it *StreamingEntryIterator) Close() error {
	if it == nil {
		return nil
	}
	if it.closed {
		return nil
	}
	it.closed = true
	if it.file != nil {
		if err := it.file.Close(); err != nil {
			return NewFileSystemError("failed to close streaming entry iterator", err)
		}
	}
	return nil
}

// GetAllEntries returns all entries (a convenience method that loads everything into memory).
func (sh *StreamingHar) GetAllEntries() ([]Entries, error) {
	if sh == nil {
		return nil, NewInvalidFormatError("StreamingHar对象为空")
	}

	var entries []Entries
	it := sh.Entries()
	for it.Next() {
		entries = append(entries, *it.Entry())
	}
	if err := it.Err(); err != nil {
		return entries, err
	}
	return entries, nil
}
