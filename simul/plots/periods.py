#!/usr/bin/env python

## This script generate the graphs that compares handel signature 
## generation with different timeouts
## TODO make also for bandwidth consumption
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

column = "sigen_wall_avg"

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
plt.ylabel("signature generation",fontsize=fs_label)
plt.xlabel("nodes",fontsize=fs_label)
plt.yscale('log')
plt.show()
