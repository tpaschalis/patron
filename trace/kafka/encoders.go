package kafka

import (
	// "github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
)

// JSONEncoder implements the Encoder interface for JSON Objects so that they can be used
// as the Key or Value in a ProducerMessage.
type JSONEncoder []byte

// ProtobufEncoder implements the Encoder interface for Protocol Buffers Objects so
// that they can be used as the Key or Value in a ProducerMessage.
type ProtobufEncoder []byte

// Encode satisfies the Encode() function of the encoder interface
func (j JSONEncoder) Encode() ([]byte, error) {
	b, err := json.Encode(j)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

// Length satisfies the Length() function of the encoder interface
func (j JSONEncoder) Length() int {
	return len(j)
}

// Encode satisfies the Encode() function of the encoder interface
func (p ProtobufEncoder) Encode() ([]byte, error) {
	b, err := protobuf.Encode(p)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

// Length satisfies the Length() function of the encoder interface
func (p ProtobufEncoder) Length() int {
	return len(p)
}

// ***********************************
// I also tried this, to dynamically set the method but it didn't work out
// ***********************************
// // EncoderObj does something
// type EncoderObj []byte

// // EncoderImpl implements the dynamic method setting for our encoder
// type EncoderImpl struct {
// 	EncoderObj EncoderObj
// 	Encode     func() ([]byte, error)
// 	Length     func() int
// }

// // SetEncoder sets the encoder
// func (e *EncoderImpl) SetEncoder(enc encoding.EncodeFunc) EncoderImpl {
// 	return EncoderImpl{
// 		Encode: func() ([]byte, error) {
// 			m, err := enc(e.EncoderObj)
// 			if err != nil {
// 				return []byte{}, err
// 			}
// 			return m, nil
// 		},
// 		Length: func() int {
// 			return len(e.EncoderObj)
// 		},
// 	}
// }
