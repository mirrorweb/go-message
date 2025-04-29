package message

import (
	"bytes"
	"io"
)

// BufferingReader reads from an underlying reader and stores the data in a buffer
// This allows the data to be re-read even after the original reader is consumed
type BufferingReader struct {
	r       io.Reader    // The underlying reader
	buffer  bytes.Buffer // Buffer to store all read data
	readEOF bool         // Whether we've reached EOF on the underlying reader
}

// NewBufferingReader creates a reader that buffers all data it reads
func NewBufferingReader(r io.Reader) *BufferingReader {
	return &BufferingReader{
		r:       r,
		buffer:  bytes.Buffer{},
		readEOF: false,
	}
}

// Read implements the io.Reader interface
func (br *BufferingReader) Read(p []byte) (n int, err error) {
	// If we've read to the end of the buffer and reached EOF, return EOF
	if br.readEOF && br.buffer.Len() == 0 {
		return 0, io.EOF
	}

	n, err = br.r.Read(p)

	// If we got data, add it to the buffer
	if n > 0 {
		br.buffer.Write(p[:n])
	}

	// If we reached EOF, mark it
	if err == io.EOF {
		br.readEOF = true
	}

	return n, err
}

// GetReader returns a new reader for the buffered data
// This reader will contain all data read so far
func (br *BufferingReader) GetReader() io.Reader {
	data := make([]byte, br.buffer.Len())
	copy(data, br.buffer.Bytes())

	return bytes.NewReader(data)
}

// RecoverableReader tries to read with a decoder first, falling back to raw content if needed
type RecoverableReader struct {
	decoder      io.Reader        // The decoder (e.g., quoted-printable, base64)
	bufReader    *BufferingReader // The buffering reader for raw content
	usingDecoder bool             // Whether we're currently using the decoder
	readSome     bool             // Whether we've read anything successfully from the decoder
}

// NewRecoverableReader creates a reader that tries to decode but falls back to raw content if needed
func NewRecoverableReader(decoder io.Reader, bufReader *BufferingReader) *RecoverableReader {
	return &RecoverableReader{
		decoder:      decoder,
		bufReader:    bufReader,
		usingDecoder: true,
		readSome:     false,
	}
}

// Read implements the io.Reader interface
func (rr *RecoverableReader) Read(p []byte) (n int, err error) {
	// Try the decoder if we're still using it
	if rr.usingDecoder {
		n, err = rr.decoder.Read(p)

		// If we read successfully, note it
		if n > 0 {
			rr.readSome = true
		}

		// If no error or just EOF after reading some data, return
		if err == nil || (err == io.EOF && rr.readSome) {
			return n, err
		}

		// If decoder failed (error other than EOF, or EOF without reading anything),
		// switch to raw reader
		rr.usingDecoder = false

		// Create a new reader from the buffered content
		rawReader := rr.bufReader.GetReader()

		// Replace the decoder with the raw reader
		rr.decoder = rawReader

		// Try reading from the raw reader
		return rr.decoder.Read(p)
	}

	// We're already using the raw reader
	return rr.decoder.Read(p)
}
