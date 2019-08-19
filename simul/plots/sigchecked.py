#!/usr/bin/env python

## This script generate the graphs that compares how many signatures
## handel checked according to different size of nodes
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

columns = {"sigs_sigCheckedCt_avg": "Average",
        "sigs_sigCheckedCt_min": "Minimum",
        "sigs_sigCheckedCt_max": "Maximum"}
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

plt.legend(fontsize=fs_label)
plt.ylabel("Number of signatures",fontsize=fs_label)
plt.xlabel("Number of nodes",fontsize=fs_label)
# plt.title("Number of signatures checked",fontsize=fs_label)
# plt.yscale('log')
plt.show()
