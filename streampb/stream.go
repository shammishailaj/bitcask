package streampb

import (
	"encoding/binary"
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	// prefixSize is the number of bytes we preallocate for storing
	// our big endian lenth prefix buffer.
	prefixSize = 8
)

// NewEncoder creates a streaming protobuf encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w, prefixBuf: make([]byte, prefixSize)}
}

// Encoder wraps an underlying io.Writer and allows you to stream
// proto encodings on it.
type Encoder struct {
	w         io.Writer
	prefixBuf []byte
}

// Encode takes any proto.Message and streams it to the underlying writer.
// Messages are framed with a length prefix.
func (e *Encoder) Encode(msg proto.Message) error {
	buf, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint64(e.prefixBuf, uint64(len(buf)))

	if _, err := e.w.Write(e.prefixBuf); err != nil {
		return errors.Wrap(err, "failed writing length prefix")
	}

	_, err = e.w.Write(buf)
	return errors.Wrap(err, "failed writing marshaled data")
}

// NewDecoder creates a streaming protobuf decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r:         r,
		prefixBuf: make([]byte, prefixSize),
	}
}

// Decoder wraps an underlying io.Reader and allows you to stream
// proto decodings on it.
type Decoder struct {
	r         io.Reader
	prefixBuf []byte
}

// Decode takes a proto.Message and unmarshals the next payload in the
// underlying io.Reader. It returns an EOF when it's done.
func (d *Decoder) Decode(v proto.Message) error {
	_, err := io.ReadFull(d.r, d.prefixBuf)
	if err != nil {
		return err
	}

	n := binary.BigEndian.Uint64(d.prefixBuf)

	buf := make([]byte, n)

	idx := uint64(0)
	for idx < n {
		m, err := d.r.Read(buf[idx:n])
		if err != nil {
			return errors.Wrap(translateError(err), "failed reading marshaled data")
		}
		idx += uint64(m)
	}
	return proto.Unmarshal(buf[:n], v)
}

func translateError(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}
