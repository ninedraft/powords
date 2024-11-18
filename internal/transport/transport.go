package transport

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

var encoding = binary.LittleEndian

type PacketKind byte

const (
	PacketChallenge PacketKind = iota
	PacketProof
	PacketData
)

func (kind PacketKind) String() string {
	switch kind {
	case PacketChallenge:
		return "PacketChallenge"
	case PacketProof:
		return "PacketProof"
	case PacketData:
		return "PacketData"
	default:
		return fmt.Sprintf("PacketKind(%d)", kind)
	}
}

var (
	errInvalidPacketKind = errors.New("invalid packet kind")
	errInvalidPacketSize = errors.New("invalid packet size")
)

type Packet struct {
	Size uint16
	Kind PacketKind
	Data []byte
}

func NewPacket(kind PacketKind, data []byte) *Packet {
	return &Packet{
		Size: uint16(len(data)),
		Kind: kind,
		Data: data,
	}
}

func (packet *Packet) WriteTo(dst io.Writer) (int64, error) {
	buf := make([]byte, 3)
	encoding.PutUint16(buf[:2], packet.Size)
	buf[2] = byte(packet.Kind)

	n, err := dst.Write(buf)
	if err != nil {
		return int64(n), fmt.Errorf("writing header: %w", err)
	}

	m, err := dst.Write(packet.Data)
	written := int64(n + m)

	if err != nil {
		return written, fmt.Errorf("writing data: %w", err)
	}

	return written, nil
}

func ReadPacket(re io.Reader) (*Packet, error) {
	buf := make([]byte, 3)
	if _, err := io.ReadFull(re, buf); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	size := encoding.Uint16(buf[:2])

	kind := PacketKind(buf[2])
	switch kind {
	case PacketChallenge, PacketProof, PacketData:
		// pass
	default:
		return nil, fmt.Errorf("invalid packet kind %s: %w", kind, errInvalidPacketKind)
	}

	data := make([]byte, size)

	if _, err := io.ReadFull(re, data); err != nil {
		return nil, fmt.Errorf("reading data: %w", err)
	}

	return &Packet{
		Size: size,
		Kind: kind,
		Data: data,
	}, nil
}

func (packet *Packet) DataChallenge() (*DataChallenge, error) {
	if packet.Kind != PacketChallenge {
		return nil, fmt.Errorf("%w: want %s, got %s", errInvalidPacketKind, PacketChallenge, packet.Kind)
	}
	if len(packet.Data) != DataChallengeSize {
		return nil, errInvalidPacketSize
	}
	return DecodeDataChallenge(packet.Data), nil
}

const SaltSize = 8

type Salt = *[SaltSize]byte

func NewSalt() Salt {
	buf := make([]byte, SaltSize)
	_, _ = io.ReadFull(rand.Reader, buf)
	return (Salt)(buf)
}

type DataChallenge struct {
	Time, Memory uint32
	KeyLen       uint32
	Difficulty   uint32
	Threads      uint8
	Salt         *[SaltSize]byte
}

const DataChallengeSize = 29

func (data *DataChallenge) Encode() []byte {
	buf := make([]byte, DataChallengeSize)

	encoding.PutUint32(buf[:4], data.Time)
	encoding.PutUint32(buf[4:8], data.Memory)
	encoding.PutUint32(buf[8:12], data.KeyLen)
	encoding.PutUint32(buf[12:20], data.Difficulty)
	buf[20] = byte(data.Threads)
	copy(buf[21:], data.Salt[:])
	return buf
}

func DecodeDataChallenge(data []byte) *DataChallenge {
	_ = data[DataChallengeSize-1]
	return &DataChallenge{
		Time:       encoding.Uint32(data[:4]),
		Memory:     encoding.Uint32(data[4:8]),
		KeyLen:     encoding.Uint32(data[8:12]),
		Difficulty: encoding.Uint32(data[12:20]),
		Threads:    data[20],
		Salt:       (Salt)(data[21:29]),
	}
}

type Conn struct {
	conn net.Conn
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{
		conn: conn,
	}
}

func (conn *Conn) Send(ctx context.Context, packet *Packet) error {
	deadline, ok := ctx.Deadline()
	if ok {
		conn.conn.SetWriteDeadline(deadline)
	}

	_, err := packet.WriteTo(conn.conn)
	return err
}

func (conn *Conn) Receive(ctx context.Context) (*Packet, error) {
	deadline, ok := ctx.Deadline()
	if ok {
		conn.conn.SetReadDeadline(deadline)
	}

	return ReadPacket(conn.conn)
}
