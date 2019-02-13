#!/usr/bin/env python

## This script generate the graphs that compares handel signature generation with different periods
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

column = "sigen_wall_avg"

files = {}
for p in [10,20,50,100]:
    f = "csv/handel_2000_%dperiod_25fail_99thr.csv" % p
    files[f] = "%dms period" % p

datas = read_datafiles(files)


for f,v in datas.items():
    x = v["totalNbOfNodes"]
    y = v[column].map(lambda x: x * 1000)
    print("file %s -> %d data points on %s" % (f,len(y),column))
    label = files[f]
    if label == "":
        label = input("Label for file %s: " % f)

    plot(x,y,"-",label,allColors.popleft())

plt.legend(fontsize=fs_label)
plt.ylabel("signature generation",fontsize=fs_label)
plt.xlabel("nodes",fontsize=fs_label)
plt.title("Signature generation time with various periods",fontsize=fs_label)
# plt.yscale('log')
plt.show()
