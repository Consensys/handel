#!/usr/bin/env python

## This script expects multiple .csv file as arguments
##
import sys
from lib import read_datafiles

import pandas as pd
import matplotlib.pyplot as plt

columns = ["sigen_wall_avg"]

files = sys.argv[1:]
datas = read_datafiles()

first = datas[files[0]]
# print(datas[first].groupby(by=columns))

x = first["totalNbOfNodes"]
y = first["sigen_wall_avg"].map(lambda x: x * 1000)

print(x)
print(y)

plt.plot(x,y)
plt.show()
