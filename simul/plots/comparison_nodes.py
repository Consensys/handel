#!/usr/bin/env python

## This script generate the graphs that compares handel, nsquare 
## and libp2p together.
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

column = "sigen_wall_avg"

# files = sys.argv[1:]
## mapping between files and label
files = {"csv/handel_0failing_99thr.csv": "handel",
        "csv/n2_4000_99thr.csv": "complete"}
        # "csv/libp2p_2000_51thr_agg1.csv": "libp2p"}
datas = read_datafiles(files.keys())


for f,v in datas.items():
    x = v["totalNbOfNodes"]
    y = v[column].map(lambda x: x * 1000)
    print("file %s -> %d data points on sigen_wall_avg" % (f,len(y)))
    label = files[f]
    if label == "":
        label = input("missing label for %s: " % f)

    print("x = ",x)
    print("y = ",y)
    plot(x,y,"-",label,allColors.popleft())

plt.legend(fontsize=fs_label)
plt.ylabel("signature generation (ms)",fontsize=fs_label)
plt.xlabel("nodes",fontsize=fs_label)
# plt.yscale('log')
plt.show()
