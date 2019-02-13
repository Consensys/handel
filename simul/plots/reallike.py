#!/usr/bin/env python

## This script generate the graphs that shows performance of handel on 
## a "real like" situation
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt
import numpy as np

# columns = {""average",
        # "sigs_sigCheckedCt_min": "minimum",
        # "sigs_sigCheckedCt_max": "maximum"}
column = "sigen_wall_avg"
#column = "net_sentBytes_avg"
# column = "sigs_sigCheckedCt_min"
# column = "sigs_sigQueueSize_avg"
# files = {"csv/handel_4000_real.csv"}
files = {"csv/handel_4000_real_2019.csv"}

datas = read_datafiles(files)

for f,v in datas.items():
    x = v["totalNbOfNodes"]
    y = v[column].map(lambda x: x*1000)
    print("file %s -> %d data points on %s" % (f,len(y),column))
    # label = files[c]
    label = "handel"
    if label == "":
        label = input("Label for file %s: " % f)

    # plot(y,y,"-",label,allColors.popleft())
    plot(x,y,"-",label,allColors.popleft())

# x = np.arange(0., 4000., 500)
# unit = 425 # for 500
# y = np.arange(0., unit * 4000 / 500, unit)
# plot(x,y,"-","linear",allColors.popleft())

plt.legend(fontsize=fs_label)
plt.ylabel("signature generation (ms)",fontsize=fs_label)
# plt.ylabel("sig queue size",fontsize=fs_label)
plt.xlabel("nodes",fontsize=fs_label)
plt.title("Handel: 75% threshold signature with 25% failings",fontsize=fs_label)
# plt.yscale('log')
plt.show()
