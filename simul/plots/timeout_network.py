#!/usr/bin/env python

## This script generates the graph that compares handel 
## bandwidth consumption with different timeouts
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

column = "net_sentBytes_avg"

files = {
        "csv/handel_2000_50timeout_99thr.csv":"50ms timeout",
        "csv/handel_2000_100timeout_99thr.csv":"100ms timeout",
        "csv/handel_2000_200timeout_99thr.csv":"200ms timeout",
        }
datas = read_datafiles(files)

for f,v in datas.items():
    x = v["totalNbOfNodes"]
    y = v[column].map(lambda x: x/1024)
    print("file %s -> %d data points on sigen_wall_avg" % (f,len(y)))
    label = files[f]
    if label == "":
        label = input("Label for file %s: " % f)

    plot(x,y,"-",label,allColors.popleft())

plt.legend(fontsize=18)
plt.ylabel("KBytes",fontsize=fs_label)
plt.xlabel("nodes",fontsize=fs_label)
# plt.yscale('log')
# plt.title("Outgoing network consumption with various timeouts",fontsize=fs_label)
plt.show()
