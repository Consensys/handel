import matplotlib.pyplot as plt
from numpy import genfromtxt
import pandas as pd
import os
import sys

destination = "../figures/"
show = True

os.environ["LC_ALL"] = "en_US.UTF-8"
os.environ["LANG"] = "en_US.UTF-8"

green = ["#557555", "#C5E1C5", "s", 10]
yellow = ["#8f8a5a", "#fffaca", "v", 11]
red = ["#8f5252", "#ffc2c2", "D", 9]
purple = ["#52528f", "#c2c2ff", "o", 10]

fs_label = 15
fs_axis = 15

ax = plt.subplot()
for label in (ax.get_xticklabels() + ax.get_yticklabels()):
    label.set_fontsize(fs_axis)

def plot(x, y, linestyle, label, color):
    plt.plot(x,y,linestyle,label= label, color=color[0], mfc=color[1],
             marker=color[2], markersize=color[3])

def save(name):
    plt.savefig(destination + name, format='eps', dpi=1000)
    if show:
        plt.show()

def save_pdf(name):
    plt.savefig(destination + name, format='pdf', bbox_inches='tight', dpi=1000)
    if show:
        plt.show()

def read_datafiles():
    if len(sys.argv) == 1:
        print("expect csv file arguments")
        sys.exit(1)

    # datas will contain all data to read organized by
    # filename : <panda's output>
    datas = {}
    for fileName in sys.argv[1:]:
        datas[fileName] = pd.read_csv(fileName)
        print("read %s : %d columns" % (fileName, len(datas[fileName].columns)))
    
    return datas

if len(sys.argv) > 1 and sys.argv[1] == "noshow":
    show = False

