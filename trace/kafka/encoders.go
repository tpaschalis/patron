package kafka

import (
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
)

// JSONEncoder represents the JSON encoder for the Kafka producer
type JSONEncoder []byte

// ProtobufEncoder represents the Protobuf encoder for the Kafka producer
type ProtobufEncoder []byte

// AvroEncoder represents the Avro encoder for the Kafka producer
type AvroEncoder []byte

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
