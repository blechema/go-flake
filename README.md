GoFlake
=========

[![Go Report Card](https://goreportcard.com/badge/github.com/blechema/go-flake?)](https://goreportcard.com/report/github.com/blechema/go-flake)

GoFlake is a distributed unique 63 bit ID generator inspired by [SONY's Sonyflake](https://github.com/sony/sonyflake) witch is inspired by [Twitter's Snowflake](https://blog.twitter.com/2010/announcing-snowflake).  

Main Goal is generating unique Ids with a hash like distribution to prevent enumeration attacks.

* Guarantees uniqueness of generated IDs over a time span of 146 years.
* 8 bit of randomness and a shuffled bit sequence generating hash like ID sequences.
* Generating of a new ID is thread save and will never block.
* Optional raw ID which are sortable like Snowflake is (but without a hash like character).
* Build in encode and decoding to and from hex, base32 and base64
* Uses 63 bit to ensure only positive values for a int64 datatype.
* Each time frame of 8.5 ms is able to hold 256 IDs. It's safe to generate unlimited more when 
stick to a cool down time of `id-count / 256 * 8.5` ms between program restarts. 
(E.g. if you generate 10,000 IDs in a row and restart the program instantly you have to wait 
at least 332ms before generating the next id)

A Flake ID is composed of

    39 bits for time units
     8 bits for sequence number
     8 bits for machine id
     8 bits for randomness

Examples of a generated ID sequence:

|  #  | Base64        | Base32          | hex                | int64                |
|:---:|:-------------:|:---------------:|:------------------:|:--------------------:|
| `1` | `COgy8KIwKF8` | `13K35S5260K5U` | `08e832f0a230285f` | `641818955994900575` |
| `2` | `Cugw8Kg6LFs` | `1BK31S5878M5M` | `0ae830f0a83a2c5b` | `785931945148820571` |
| `3` | `Cuow-qowLl0` | `1BL31ULA60N5Q` | `0aea30faaa302e5d` | `786494938084814429` |
| `4` | `COo68KI8Klk` | `13L3LS527GL5I` | `08ea3af0a23c2a59` | `642390702042131033` |

Installation
------------

```
go get github.com/blechema/go-flake
```

Usage
-----

The function `Next()` generates a new unique ID utilizing the Default generator singleton (intended for a single instance installations).

```go
func Next() Flake
```

The function `Raw()` generates a raw ID as int64 utilizing the Default generator singleton.

```go
func NextRaw() int64
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
