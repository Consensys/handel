#!/usr/bin/env python

## This script generate the graphs that compares handel signature 
## generation with different number of failing nodes for a fixed 
## number of total nodes, and a fixed threshold 51%
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

netColumn = "net_sentBytes_avg"
nodeColumn = "totalNbOfNodes"
failingColumn = "failing"

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
    y = v[netColumn].map(lambda x: x/1024)
    print("file %s -> %d data points on %s" % (f,len(y),netColumn))
    label = files[f]
    if label == "":
        label = input("Label for file %s: " % f)

    plot(x,y,"-",label,allColors.popleft())

plt.legend(fontsize=fs_label)
plt.ylabel("KBytes ",fontsize=fs_label)
plt.xlabel("failing nodes in %",fontsize=fs_label)
# plt.yscale('log')
# plt.title("Outgoing network consumption for 51% signature threshold over 4000 nodes") 
plt.show()
