#!/usr/bin/env python

## This script generates the graph that compares handel signature
## generation time with different thresholds
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

column = "sigen_wall_avg"

files = {
        "csv/handel_0failing_51thr.csv":"51% threshold",
        "csv/handel_0failing_75thr.csv":"75% threshold",
        "csv/handel_0failing_99thr.csv":"99% threshold",
        }
datas = read_datafiles(files)

for f,v in datas.items():
    x = v["totalNbOfNodes"]
    y = v[column].map(lambda x: x * 1000)
    print("file %s -> %d data points on sigen_wall_avg" % (f,len(y)))
    label = files[f]
    if label == "":
        label = input("Label for file %s: " % f)

    plot(x,y,"-",label,allColors.popleft())

plt.legend(fontsize=fs_label)
plt.ylabel("signature generation (ms)",fontsize=fs_label)
plt.xlabel("nodes",fontsize=fs_label)
plt.title("signature generation time with various thresholds",fontsize=fs_label)
# plt.yscale('log')
plt.show()
