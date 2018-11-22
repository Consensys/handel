Handel nodes keeps each intermediate multi-signature separate:
    - for each level there is a "partial" multisignature associated of the
      expected bitset's length, i.e. for level 2, the length of the bitset is 2.
    - to send a signature of a given level l, one needs to combine all
      independant multi-signatures it has below level l. For example:
        + there are 8 nodes, and we consider the point of view of node 1
        + node 1 needs to send the signatures to nodes corresponding to level 3
          (the last step, containing the other half of the nodes)
        + node 1 will create a level 3 signature, of length 4, by
          "concatenating" all individual signatures of level 0,1, 2 and 3.
          (level 0 is the node's 1 signature)
        -> we call this combination a "combined" signature

Handel keeps track of the level it has already sent signatures to the
corresponding nodes.  When starting a new level, handel does the following:
    - Take the combination of all stored partial signatures up until the the
      current level started *excluded*, and sends it to the nodes of the new
      current level of handel. When the current level is the maximum, this
      routine never gets executed anymore.


Rules:
    - Once we receive a signature of a given level that is *complete*, we start
      the next level, if not started already.
    - Once we receive a signature of a given level, we see if it improves the
      combined signature to send to the next level. If it does (and the
      cardinality is superior to the threshold), we send it.
    - 
