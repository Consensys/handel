#!/usr/bin/env python

## This script generate the graphs that compares how many signatures
## handel checked according to different size of nodes
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

columns = {"sigs_sigCheckedCt_avg": "average",
        "sigs_sigCheckedCt_min": "minimum",
        "sigs_sigCheckedCt_max": "maximum"}
files = {"csv/handel_0failing_99thr.csv": "handel"}

datas = read_datafiles(files)


for f,v in datas.items():
    x = v["totalNbOfNodes"]
    for c in columns:
        y = v[c]
        print("file %s -> %d data points on %s" % (f,len(y),c))
        label = columns[c]
        if label == "":
            label = input("Label for file %s: " % f)

        plot(x,y,"-",label,allColors.popleft())

plt.legend(fontsize=18)
plt.ylabel("signatures checked",fontsize=fs_label)
plt.xlabel("nodes",fontsize=fs_label)
# plt.title("Number of signatures checked",fontsize=fs_label)
# plt.yscale('log')
plt.show()
