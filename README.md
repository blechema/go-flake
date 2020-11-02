GoFlake
=========

[![Go Report Card](https://goreportcard.com/badge/github.com/blechema/go-flake?)](https://goreportcard.com/report/github.com/blechema/go-flake)

GoFlake is a distributed unique 63 bit ID generator inspired by [SONY's Sonyflake](https://github.com/sony/sonyflake) witch is inspired by [Twitter's Snowflake](https://blog.twitter.com/2010/announcing-snowflake).  

Main Goal is generating unique Ids with a hash like distribution to prevent enumeration attacks.

* Guarantees uniqueness of generated IDs over a time span of 146 years.
* 16 bit of randomness and a shuffled bit sequence generating hash like ID sequences.
* Supports 256 different machines
* Generating of a new ID is thread save and will never block.
* Optional raw ID which are sortable like Snowflake is (but with less hash like character).
* Build in encoding and decoding to and from hex, base32 and base64
* Uses 63 bit to ensure positive values for an int64 datatype.
* Each one second time frame is capable to hold more than 4,000,000 IDs. It's safe to generate unlimited more when 
stick to a cool down time of `id-count / 4,000,000` seconds between program restarts.

A Flake ID is composed of

* 4 bytes of time
* 1 byte sequence counter  
* 2 random bytes
* 1 byte machine id

The amount of random bytes will decrease to 1 byte when more than 64 Ids generated within a timespan of one second.
The random bytes will turn off when more than 8,224 Ids generated within a timespan of one second.

Examples of a generated ID sequence:

|  #  | Base64        | Base32          | hex                | int64                |
|:---:|:-------------:|:---------------:|:------------------:|:--------------------:|
| `1` | `QDBAQEBwAAE` | `80O40G20E0002` | `4030404040700001` | `4625267462012665857` |
| `2` | `RjRGRkB0Bg8` | `8OQ4CHI0EG30U` | `463446464074060f` | `5058745548986910223` |
| `3` | `QDBEQEBwCgM` | `80O48G20E0506` | `4030444040700a03` | `4625271860059179523` |
| `4` | `QjRCREZ2Cgk` | `88Q44H26EO50I` | `4234424446760a09` | `4770510766299548169` |

Installation
------------

```
go get github.com/blechema/go-flake
```

Usage
-----

The function `Next()` generates a new unique ID utilizing the `Default` generator singleton (intended for a single instance installation).

```go
func Next() Flake
```

The function `NextRaw()` generates a raw ID as int64 utilizing the `Raw` generator singleton (intended for a single instance installation).

```go
func NextRaw() Flake
```

Create a custom generator instance to define machine-id and an epoch start time.

```go
flaker := Default.WithMachineId(123).WithEpochStart(time.Unix(1604160000, 0))
id := flaker.Next()
```

Integrated encoding and decoding.

```go
hex := id.Hex()
base32 := id.Base32()
base64 := id.Base64()

id1, err := Decode(hex)
id2, err := Decode(base32)
id3, err := Decode(base64)
```

Flake derives from `int64` so conversion can be done simply: `id := int64(flake)`.

License
-------

The MIT License

See [LICENSE](https://github.com/blechema/go-flake/main/LICENSE) for details. 
