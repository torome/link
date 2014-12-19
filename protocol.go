package link

import (
	"encoding/binary"
	"io"
)

var (
	BigEndian    = binary.BigEndian
	LittleEndian = binary.LittleEndian
)

// Packet spliting protocol.
// You can implement custom packet protocol for special protocol.
type Protocol interface {
	// Packet a message into buffer. The buffer maybe grows.
	Packet(buffer *Buffer, message Message) error

	// Write a packet. The buffer maybe grows.
	Write(writer io.Writer, buffer *Buffer) error

	// Read a packet. The buffer maybe grows.
	Read(reader io.Reader, buffer *Buffer) error
}

// The packet spliting protocol like Erlang's {packet, N}.
// Each packet has a fix length packet header to present packet length.
type SimpleProtocol struct {
	n             int
	bo            binary.ByteOrder
	head          []byte
	encodeHead    func([]byte)
	decodeHead    func() int
	MaxPacketSize int
}

// Create a {packet, N} protocol.
// The n means how many bytes of the packet header.
func PacketN(n int, byteOrder binary.ByteOrder) *SimpleProtocol {
	protocol := &SimpleProtocol{
		n:    n,
		bo:   byteOrder,
		head: make([]byte, n),
	}

	switch n {
	case 1:
		protocol.encodeHead = func(buffer []byte) {
			buffer[0] = byte(len(buffer) - n)
		}
		protocol.decodeHead = func() int {
			return int(protocol.head[0])
		}
	case 2:
		protocol.encodeHead = func(buffer []byte) {
			byteOrder.PutUint16(buffer, uint16(len(buffer)-n))
		}
		protocol.decodeHead = func() int {
			return int(byteOrder.Uint16(protocol.head))
		}
	case 4:
		protocol.encodeHead = func(buffer []byte) {
			byteOrder.PutUint32(buffer, uint32(len(buffer)-n))
		}
		protocol.decodeHead = func() int {
			return int(byteOrder.Uint32(protocol.head))
		}
	case 8:
		protocol.encodeHead = func(buffer []byte) {
			byteOrder.PutUint64(buffer, uint64(len(buffer)-n))
		}
		protocol.decodeHead = func() int {
			return int(byteOrder.Uint64(protocol.head))
		}
	default:
		panic("unsupported packet head size")
	}

	return protocol
}

// Write a packet. The buffer maybe grows.
func (p *SimpleProtocol) Packet(buffer *Buffer, message Message) error {
	size := message.RecommendBufferSize()
	if cap(buffer.Data) < size {
		buffer.Data = make([]byte, p.n, size)
	} else {
		buffer.Data = buffer.Data[:p.n]
	}
	return message.WriteBuffer(buffer)
}

// Write a packet. The buffer maybe grows.
func (p *SimpleProtocol) Write(writer io.Writer, buffer *Buffer) error {
	if p.MaxPacketSize > 0 && len(buffer.Data) > p.MaxPacketSize {
		return PacketTooLargeError
	}

	p.encodeHead(buffer.Data)

	if _, err := writer.Write(buffer.Data); err != nil {
		return err
	}

	return nil
}

// Read a packet. The buffer maybe grows.
func (p *SimpleProtocol) Read(reader io.Reader, buffer *Buffer) error {
	if _, err := io.ReadFull(reader, p.head); err != nil {
		return err
	}

	size := p.decodeHead()

	if p.MaxPacketSize > 0 && size > p.MaxPacketSize {
		return PacketTooLargeError
	}

	if cap(buffer.Data) < size {
		buffer.Data = make([]byte, size)
	} else {
		buffer.Data = buffer.Data[0:size]
	}

	if size == 0 {
		return nil
	}

	if _, err := io.ReadFull(reader, buffer.Data); err != nil {
		return err
	}

	return nil
}
