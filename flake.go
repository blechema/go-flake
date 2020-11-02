package flake

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net"
	"sync"
	"time"
)

// Flake represents a unique 63 bit ID.
type Flake int64

// Flaker is the generator interface.
type Flaker interface {
	Next() Flake
	WithMachineId(machineId byte) Flaker
	WithEpochStart(time time.Time) Flaker
}

// ----------------------------------------------------------------------------

type flaker struct {
	mutex           *sync.Mutex
	raw             bool
	machineId       byte
	epochStart      int64
	sequence        int32
	currentInterval int64
}

// [interval(4byte)][sequence/random(3byte)][machine(1byte)]
const (
	intervalBits    = 32
	sequenceBits    = 23
	machineIdBits   = 8
	ignoredTimeBits = 30
	intervalMask    = (1 << intervalBits) - 1
	machineIdMask   = (1 << machineIdBits) - 1
)

var base32RawEncoding = base32.HexEncoding.WithPadding(base32.NoPadding)

// Default is the default singleton of Flaker with sets the lower 8 bits of
// the first non loopback IPv4 address (zero if not available) as machine-id
// and the 1/1/2020 as epoch start (epoch is only needed for sortable IDs).
var Default = Flaker(&flaker{
	mutex:      &sync.Mutex{},
	machineId:  byte(getLocalIPv4() & machineIdMask),
	epochStart: 1577833200000000000, // 1/1/2020
})

var Raw = Flaker(&flaker{
	raw:        true,
	mutex:      &sync.Mutex{},
	machineId:  byte(getLocalIPv4() & machineIdMask),
	epochStart: 1577833200000000000, // 1/1/2020
})

// ----------------------------------------------------------------------------

// Next is a shorthand for Default.Next()
func Next() Flake {
	return Default.Next()
}

// NextRaw is a shorthand for Raw.Next()
func NextRaw() Flake {
	return Raw.Next()
}

// WithMachineId is a shorthand for Default.WithMachineId(machineId)
func WithMachineId(machineId byte) Flaker {
	return Default.WithMachineId(machineId)
}

// WithEpochStart is a shorthand for Default.WithEpochStart(time)
func WithEpochStart(time time.Time) Flaker {
	return Default.WithEpochStart(time)
}

// ----------------------------------------------------------------------------

// Returns a new unique ID in shuffled bits flake-format. Flake derives
// actually from an int64 so you can convert with int64(flake). The IDs will
// be guarantied unique within a 146 years time span. It can generate up to
// 4,000,000 IDs each second but its save to generate unlimited more when stick
// to a cool down time of GENERATED_IDS / 4,000,000 s between program restarts.
// Generating a new ID is thread save and will never block.
func (g *flaker) Next() Flake {

	raw := g.next()

	if g.raw {
		return Flake(raw)
	}

	// Shuffle bits
	uid := make([]byte, 8, 8)
	for i := int64(0); i < 8; i++ {
		for l := int64(0); l < 8; l++ {
			uid[l] |= byte((raw & (1 << (i*8 + l))) >> (i*7 + l))
		}
	}

	return Flake(binary.LittleEndian.Uint64(uid))
}

// next returns a raw unique ID generated from the flake algorithm but without
// shuffled bits. This representation of a unique ID is sortable and
// will increasing until end of flake epoch (2116-02-21) when the
// sequence will start again. No matter that the IDs will be guarantied
// unique within a 146 years time span. Generating a new ID is thread save
// and will never block.
func (g *flaker) next() int64 {

	// 32 bit time interval with nano-time >> 20 (~1s) clock loops after reaching end of epoch each ~ 146 years
	interval := ((time.Now().UnixNano() - g.epochStart) >> ignoredTimeBits) & intervalMask

	// 23 bit sequence and random
	sequence := int32(0)
	g.mutex.Lock()
	loop := (g.sequence + 0x400000 - 0x2020) >> sequenceBits // 4194304 - 8224 = 4186080
	if interval-int64(loop) <= g.currentInterval {
		g.sequence++
		if g.sequence < 0x20 {
			// Small counter and 2 random bytes
			sequence = (g.sequence << 16) | (randomByte() << 8) | randomByte()
		} else if g.sequence < 0x2020 {
			// Enlarge the counter
			sequence = (0x200000 - 0x2000 + (g.sequence << 8)) | randomByte()
		} else {
			// Use all space for the counter
			sequence = 0x400000 - 0x2020 + g.sequence
		}
	} else {
		g.currentInterval = interval
		g.sequence = int32(0)
	}
	g.mutex.Unlock()

	raw := interval
	raw = (raw << sequenceBits) + int64(sequence) // + to increment the interval too on rollover
	raw = (raw << machineIdBits) | int64(g.machineId)

	return raw
}

// Returns a new Flaker instance copy with the specified machine-id set. You
// should create one Flaker instance per machine as singleton. Do not create
// multiple instances with the same machine-id since it's not guarantied to
// generate unique IDs from different instances with the same machine-id.
func (g flaker) WithMachineId(machineId byte) Flaker {
	g.machineId = machineId
	g.mutex = &sync.Mutex{}
	return &g
}

// Returns a new Flaker instance copy with the specified epoch start time set.
// A flaker epoch will last 146 years. The generated IDs will be guarantied
// unique within this time span. You don't have to set this value as long you
// don't need sorted ID values generated with the NextRaw() function. The uniqueness
// of the generated IDs is guarantied within a timespan of 146 years anyhow.
func (g flaker) WithEpochStart(time time.Time) Flaker {
	g.epochStart = time.UnixNano()
	g.mutex = &sync.Mutex{}
	return &g
}

// ----------------------------------------------------------------------------

// Bytes returns the flak as 8 bytes
func (f Flake) Bytes() []byte {
	uid := make([]byte, 8, 8)
	binary.BigEndian.PutUint64(uid, uint64(f))
	return uid
}

// Int64 returns the flak as raw int64
func (f Flake) Int64() int64 {
	return int64(f)
}

// Uint64 returns the flak as raw uint64
func (f Flake) Uint64() uint64 {
	return uint64(f)
}

// Hex encodes the flake to hex
func (f Flake) Hex() string {
	return hex.EncodeToString(f.Bytes())
}

// Base64 encodes the flake to base64
func (f Flake) Base64() string {
	return base64.RawURLEncoding.EncodeToString(f.Bytes())
}

// Base32 encodes the flake to base32
func (f Flake) Base32() string {
	return base32RawEncoding.EncodeToString(f.Bytes())
}

// FromBytes decodes a 8 bit flake instance from bytes
func FromBytes(b []byte) (flake Flake, err error) {
	if len(b) != 8 {
		return 0, errors.New("unknown format")
	}
	flake = Flake(binary.BigEndian.Uint64(b))
	return
}

// Decode decodes a hex, base32 or base64 encoded flake
func Decode(s string) (flake Flake, err error) {
	var b []byte
	switch len(s) {
	case 11:
		b, err = base64.RawURLEncoding.DecodeString(s)
	case 13:
		b, err = base32RawEncoding.DecodeString(s)
	case 16:
		b, err = hex.DecodeString(s)
	default:
		err = errors.New("unknown format")
	}
	if err != nil {
		return
	}
	return FromBytes(b)
}

// ----------------------------------------------------------------------------

func randomByte() int32 {
	b := make([]byte, 1, 1)
	_, _ = rand.Read(b)
	return int32(b[0])
}

func getLocalIPv4() (ip4 uint32) {
	addrs, _ := net.InterfaceAddrs()
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok {
			if ip := ipnet.IP.To4(); ip != nil && !ip.IsLoopback() &&
				(ip[0] == 10 || ip[0] == 172 && (ip[1] >= 16 && ip[1] < 32) || ip[0] == 192 && ip[1] == 168) {
				ip4 = binary.BigEndian.Uint32(ip)
				break
			}
		}
	}
	return
}
