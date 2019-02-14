#!/usr/bin/env python

## This script generate the graphs that compares handel signature 
## generation with different number of failing nodes for a fixed 
## number of total nodes, and a fixed threshold 51%
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

sigColumn = "sigen_wall_avg"
nodeColumn = "totalNbOfNodes"
failingColumn = "failing"

yColumns = {"sigen_wall_min": "Minimum",
            "sigen_wall_avg": "Average",
            "sigen_wall_max": "Maximum"}
            

## threshold of signatures required
threshold = "51"
expectedNodes = 4000
nodes = None

files = {"csv/handel_4000_failing.csv": "handel"}
datas = read_datafiles(files)

for f,v in datas.items():
    nodes = v[nodeColumn].max() # should be 2000
    if int(v[nodeColumn].mean()) != expectedNodes:
        print("error : nodes should be " + str(expectedNodes))
        sys.exit(1)

    x = v[failingColumn].map(lambda x: int((x/nodes) * 100))
    for c,name in yColumns.items():
        y = v[c]
        print("file %s -> %d data points on %s" % (f,len(y),sigColumn))
        # label = files[f]
        label = name
        if label == "":
            label = input("Label for file %s: " % f)

        plot(x,y,"-",label,allColors.popleft())

plt.legend(fontsize=18)
plt.ylabel("signature generation (ms)",fontsize=fs_label)
plt.xlabel("failing nodes in %",fontsize=fs_label)
# plt.yscale('log')
# plt.title("Time for 51% signature threshold over 4000 nodes")
# plt.axis('square')
plt.show()
