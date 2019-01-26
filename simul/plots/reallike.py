#!/usr/bin/env python

## This script generate the graphs that shows performance of handel on 
## a "real like" situation
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

# columns = {""average",
        # "sigs_sigCheckedCt_min": "minimum",
        # "sigs_sigCheckedCt_max": "maximum"}
column = "sigen_wall_avg"
files = {"csv/handel_4000_real.csv"}

datas = read_datafiles(files)


for f,v in datas.items():
    x = v["totalNbOfNodes"]
    y = v[column].map(lambda x: x*1000)
    print("file %s -> %d data points on %s" % (f,len(y),column))
    # label = files[c]
    label = "handel"
    if label == "":
        label = input("Label for file %s: " % f)

    plot(x,y,"-",label,allColors.popleft())

plt.legend(fontsize=fs_label)
plt.ylabel("signature generation (ms)",fontsize=fs_label)
plt.xlabel("nodes",fontsize=fs_label)
plt.title("Handel: 75% threshold signature with 25% failings",fontsize=fs_label)
# plt.yscale('log')
plt.show()
