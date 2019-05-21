[![CircleCI](https://circleci.com/gh/ConsenSys/handel.svg?style=svg)](https://circleci.com/gh/ConsenSys/handel)

# Handel

Handel is a fast multi-signature aggregation protocol for large Byzantine
committees. This is the reference implementation in Go.

You can find the [slides](https://docs.google.com/presentation/d/1fL0mBF5At4ojW0HhbvBQ2yJHA3_q8q8kiioC6WvY9g4/edit?usp=sharing) presented at [Stanford Blockchain Conference 2019](https://cyber.stanford.edu/sbc19)

This implementation was used to demonstrate the results presented at SBC19, aggregating BLS signatures on 4000 hande nodes.
Its designed to be easily extensible, with interfaces for add aggregations methods, curves, etc.

