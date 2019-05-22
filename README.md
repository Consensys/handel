[![CircleCI](https://circleci.com/gh/ConsenSys/handel.svg?style=svg)](https://circleci.com/gh/ConsenSys/handel)

# Handel

Handel is a fast multi-signature aggregation protocol for large Byzantine
committees. This is the reference implementation in Go.

## The protocol

Handel is a Byzantine fault tolerant aggregation protocol that allows for the
quick aggregation of cryptographic signatures over a WAN.  Handel has both
logarithmic time and network complexity and needs minimal computing resources.
For more information about the protocol, we refer you to the following
presentations:
+ [Stanford Blockchain Conference 2019](https://cyber.stanford.edu/sbc19): the [slides](https://docs.google.com/presentation/d/1fL0mBF5At4ojW0HhbvBQ2yJHA3_q8q8kiioC6WvY9g4/edit?usp=sharing) presented.
+ [Community Ethereum Development Conference 2019](https://www.edcon.io/): the
  [slides](https://pandax-statics.oss-cn-shenzhen.aliyuncs.com/statics/1225469768899493.pdf)

We have a paper in submission which should be released to the public soon.
Please note that the slides are not up-to-date with the latest version of the
paper.

## The reference implementation 

Handel is an open-source Go library implementing the protocol. It includes many
openness points to allow plugging different signature schemes or even other
forms of aggregation besides signature aggregation. We implemented extensions to
use Handel with BLS multi-signatures using the BN256 curve.  We ran large-scale
tests and evaluated Handel on 2000 AWS nano instances located in 10 AWS regions
and running two Handel nodes per instance. Our results show that Handel scales
logarithmically with the number of nodes both in communication and re- source
consumption. Handel aggregates 4000 BN256 signatures with an average of 900ms
completion time and an average of 56KBytes network consumption.

If you want to hack around the library, you can find more information about the
internal structure of Handel in the
[HACKING.md](https://github.com/consensys/handel/blob/master/HACKING.md) file.

## License

The library is licensed with an Apache 2.0 license. See
[LICENSE](https://github.com/consensys/handel/blob/master/LICENSE) for more
information.
