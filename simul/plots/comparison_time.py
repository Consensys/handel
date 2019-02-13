#!/usr/bin/env python

## This script generate the graphs that compares handel, nsquare 
## and libp2p together for the signature generation time
##
import sys
from lib import *

import pandas as pd
import matplotlib.pyplot as plt

fs_label = 22
column = "sigen_wall_avg"

# files = sys.argv[1:]
## mapping between files and label
files = {"csv/handel_0failing_99thr.csv": { 
            "label": "handel",
            "xOffset": 0.15,
            "yOffset": 0.09,
        },
        "csv/n2_4000_99thr.csv": {
            "label": "complete",
            "xOffset": 0.20,
            "yOffset": 0.01,
        }
        }
# "csv/libp2p_2000_51thr_agg1.csv": "libp2p"}
datas = read_datafiles(files.keys())


for f,v in datas.items():
    x = v["totalNbOfNodes"]
    y = v[column].map(lambda x: x * 1000)
    print("file %s -> %d data points on sigen_wall_avg" % (f,len(y)))
    label = files[f]["label"]
    if label == "":
        label = input("missing label for %s: " % f)

    print("x = ",x)
    print("y = ",y)
    plot(x,y,"-",label,allColors.popleft())
    xMax = x.max()
    yMax = y.max()
    xOffset = files[f]["xOffset"]
    yOffset = files[f]["yOffset"]
    xCoordText = xMax - (xMax * xOffset) 
    ## 10% higher
    yCoordText = yMax + (yMax * yOffset)
    plt.annotate("%d ms" % yMax,xy=(xMax,yMax),xycoords='data',
            xytext=(xCoordText,yCoordText),textcoords='data',fontsize=fs_label)


plt.legend(fontsize=fs_label)
plt.ylabel("signature generation (ms)",fontsize=fs_label)
plt.xlabel("Handel nodes",fontsize=fs_label)
# plt.title("Signature generation time - comparative baseline",fontsize=fs_label)
# plt.yscale('log')
plt.show()
