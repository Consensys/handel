# Strategy for simulation

## Architecture

How do we collect measurements from multiple running Handel instances ?


## Implementation

We need to have several components implemented:
+ a Network implementation using UDP / TCP 
+ a file-based registry that can read from a CSV file for example
+ configurable network, store, processing and Handel so we can wrap them around
  measurements. Multiple solutions possible:
  1. Export Handel's field of general interfaces (processing, store etc) so one
     can wrap some into another interface. simul/ package can contain the
     wrappers.
     + PRO: Very easy to wrap interfaces around 
     + CON: Public field of handels exposed
  2. Having "constructor" function for each interface that are put into the
     config struct. Can even put public the current implementations.
     + PRO: quite modularizable.
     + CON: larger config, difficult to know in advance which fields are
       required when creating an interface
  3. sets up a "SimulationHandel" struct, with its own interfaces inside the
     handel package
     + PRO: every implementation details could be kept hidden but still usable
       for collecting results, code should be able to be separated from main
       logic
     + CON: simulation code separated but still in same package, not so
       "production-ready".
