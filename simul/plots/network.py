#!/usr/bin/env python

## This script generate the graphs that compares handel, nsquare 
## and libp2p together.
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

column = "net_sentBytes_avg"

files = sys.argv[1:]
datas = read_datafiles()


for f,v in datas.items():
    x = v["totalNbOfNodes"]
    y = v[column]
    print("file %s -> %d data points on %s" % (f,len(y),column))
    label = input("Label for file %s: " % f)
    if label == "":
        label = f

    plot(x,y,"-",label,allColors.popleft())

plt.legend(fontsize=fs_label)
plt.ylabel("bytes sent (avg)",fontsize=fs_label)
plt.xlabel("nodes",fontsize=fs_label)
plt.yscale('log')
plt.show()
