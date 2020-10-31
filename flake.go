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

type Flake int64

type Flaker interface {
	Next() Flake
	NextRaw() int64
	WithMachineId(machineId byte) Flaker
	WithEpochStart(time time.Time) Flaker
}

// ----------------------------------------------------------------------------

type flaker struct {
	mutex           *sync.Mutex
	machineId       byte
	epochStart      int64
	sequence        int64
	currentInterval int64
	runner          byte
}

const (
	timeIntervalBits    = 39
	sequenceCounterBits = 8
	machineIdBits       = 8
	randomBits          = 8
	ignoredTimeBits     = 23
	timeIntervalMask    = (1 << timeIntervalBits) - 1
	machineIdMask       = (1 << machineIdBits) - 1
	randomMask          = (1 << randomBits) - 1
)

var base32RawEncoding = base32.HexEncoding.WithPadding(base32.NoPadding)

// The default singleton of Flaker with sets the lower 8 bits of the first
// non loopback IPv4 address (zero if not available) as machine-id and
// the 1/1/2020 as epoch start (epoch is only needed for sortable IDs).
var Default = Flaker(&flaker{
	mutex:      &sync.Mutex{},
	machineId:  byte(getLocalIPv4() & machineIdMask),
	epochStart: 1577833200000000000, // 1/1/2020
})

// ----------------------------------------------------------------------------

// Shorthand for Default.Next()
func Next() Flake {
	return Default.Next()
}

// Shorthand for Default.NextRaw()
func NextRaw() int64 {
	return Default.NextRaw()
}

// Shorthand for Default.WithMachineId(machineId)
func WithMachineId(machineId byte) Flaker {
	return Default.WithMachineId(machineId)
}

func WithEpochStart(time time.Time) Flaker {
	return Default.WithEpochStart(time)
}

// ----------------------------------------------------------------------------

// Returns a new unique ID in shuffled bits flake-format. Flake derives
// actually from an int64 so you can convert with int64(flake). The IDs will
// be guarantied unique within a 146 years time span. It can generate up to
// 256 IDs each 8.5 ms but its save to generate much more when stick to a
// cool down time of GENERATED_IDS * 8.5 / 256 ms between program restarts
// (or between creating new Flaker instances - with will not be a good
// practice anyhow). Generating a new ID is thread save and will never block.
func (g *flaker) Next() Flake {

	raw := g.NextRaw()

	// Shuffle bits
	uid := make([]byte, 8, 8)
	for i := int64(0); i < 8; i++ {
		for l := int64(0); l < 8; l++ {
			uid[l] |= byte((raw & (1 << (i*8 + l))) >> (i*7 + l))
		}
	}

	return Flake(binary.LittleEndian.Uint64(uid))
}

// Returns a raw unique ID generated from the flake algorithm but without
// shuffled bits. This representation of a unique ID is sortable and
// will increasing until end of flake epoch (2116-02-21) when the
// sequence will start again. No matter that the IDs will be guarantied
// unique within a 146 years time span. Generating a new ID is thread save
// and will never block.
func (g *flaker) NextRaw() int64 {

	// 39 bit time interval with nano-time >> 23 (~8ms) clock loops after reaching end of epoch each ~ 150 years
	interval := ((time.Now().UnixNano() + g.epochStart) >> ignoredTimeBits) & timeIntervalMask

	// 8 bit counter
	g.mutex.Lock()
	g.runner++
	loop := g.sequence >> sequenceCounterBits
	if interval-loop <= g.currentInterval {
		g.sequence++
	} else {
		g.currentInterval = interval
		g.sequence = 0
	}
	sequence := (loop << 8) + int64(g.runner)
	g.mutex.Unlock()

	// 8 bit of randomness
	b := make([]byte, 1, 1)
	_, _ = rand.Read(b)
	random := int64(b[0]) & randomMask

	// Build 63 bit raw result => [interval][sequence][random][machine]
	raw := interval
	raw = (raw << sequenceCounterBits) + sequence // + to increment the interval too on rollover
	raw = (raw << randomBits) | random
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

// Returns the flak as 8 bytes
func (f Flake) Bytes() []byte {
	uid := make([]byte, 8, 8)
	binary.BigEndian.PutUint64(uid, uint64(f))
	return uid
}

// Returns the flak as raw int64
func (f Flake) Int64() int64 {
	return int64(f)
}

// Returns the flak as raw uint64
func (f Flake) Uint64() uint64 {
	return uint64(f)
}

// Encodes the flake to hex
func (f Flake) Hex() string {
	return hex.EncodeToString(f.Bytes())
}

// Encodes the flake to base64
func (f Flake) Base64() string {
	return base64.RawURLEncoding.EncodeToString(f.Bytes())
}

// Encodes the flake to base32
func (f Flake) Base32() string {
	return base32RawEncoding.EncodeToString(f.Bytes())
}

// Decodes a 8 bit flake instance from bytes
func FromBytes(b []byte) (flake Flake, err error) {
	if len(b) != 8 {
		return 0, errors.New("unknown format")
	}
	flake = Flake(binary.BigEndian.Uint64(b))
	return
}

// Decodes a hex, base32 or base64 encoded flake
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
