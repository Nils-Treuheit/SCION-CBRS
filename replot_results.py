import matplotlib.pyplot as plt
from sys import argv
import pandas as pd
import numpy as np
from enum import Enum
from glob import glob
from os.path import abspath

class ResFromat(Enum):
    A4_half = 0
    A4_full = 1
    HD24 = 2
    UHD32 = 3

COLORS = ["black", 
          "gold", 
          "lightcoral", 
          "brown", 
          "red", 
          "turquoise", 
          "tab:blue",
          "orange",
          "lime",
          "darkviolet",
          "dodgerblue"]

legend = True if len(argv)>5 and int(argv[5])>0 else False
overwrite = True if len(argv)>4 and int(argv[4])>0 else False
recursive = True if len(argv)>3 and int(argv[3])>0 else False
res_format = ResFromat(int(argv[2])) if len(argv)>2 else ResFromat.UHD32
if len(argv)>1: 
    directory = argv[1]
    files = glob(abspath(directory)+"/*.csv")
    if recursive: files += glob(abspath(directory)+"/**/*.csv")

    for idx,filename in enumerate(files):
        data = pd.read_csv(filename, index_col=0)
        data.index = range(len(data.index))
        fig = plt.figure(idx,figsize=(20, 11)) 
        ax = fig.gca()

        for id,col in enumerate(data.columns):
            fetches = len(data[col])
            successfull = fetches-data[col].isna().sum()
            fetch_times = data[col][~data[col].isna()] 
            print("\nFetching",col,"results:")
            print("=> {:.4f}% of all URL fetches have been successfull.".format(successfull/fetches * 100))
            print("=> The average fetch time was {:.9f} seconds".format(np.mean(fetch_times)))
            print("=> The median fetch time was {:.9f} seconds".format(np.median(fetch_times)))
            print("=> The standard deviation was {:.6f}".format(np.std(fetch_times)))
            print("=> The fastest fetch took {:.9f}".format(np.min(fetch_times)))
            print("=> The slowest fetch took {:.9f}".format(np.max(fetch_times)))
            ax.plot(fetch_times,c=COLORS[id],label=col.strip("/").split("/")[-1].strip("www."))

        fig.suptitle(((("           " if not legend else "")+"Parallel ") if "par" in filename else (("         " if not legend else "")+"Sequential "))+"Fetching Stats")
        ax.set_title("URL Fetch Times from"+(f"\n{data.columns[0]}" if res_format == ResFromat.A4_half else f" {data.columns[0]}" ))
        ax.set_xlabel("Fetch ID")
        ax.set_ylabel("Time (sec)")
        ax.set_yscale('log')
        if legend: ax.legend(loc='upper left', bbox_to_anchor=(1.02, 1))
        fig.tight_layout()
        name = ((filename.split("fetch_times")[0]+"fetch_stats") if "fetch_times" in filename else (filename.split(".")[0]))
        if res_format == ResFromat.A4_half:
            fig.set_size_inches(4,3)
            if not legend:
                fig.subplots_adjust(left=0.18, right=0.96, top=0.78, bottom=0.16)
            else:
                fig.subplots_adjust(left=0.18, right=0.70, top=0.78, bottom=0.16)
            fig.savefig((name if overwrite else (name+'_half_A4.png')), dpi=100)
        elif res_format == ResFromat.A4_full:
            fig.set_size_inches(8,4)
            if not legend:
                fig.subplots_adjust(left=0.09, right=0.98, top=0.87, bottom=0.13)
            else:
                fig.subplots_adjust(left=0.09, right=0.72, top=0.87, bottom=0.13)
            fig.savefig((name if overwrite else (name+'_full_A4.png')), dpi=200)
        elif res_format == ResFromat.HD24:
            fig.set_size_inches(20.3,11.5)
            if not legend:
                fig.subplots_adjust(left=0.06, right=0.98, top=0.94, bottom=0.06)
            else:
                fig.subplots_adjust(left=0.06, right=0.72, top=0.94, bottom=0.06)
            fig.savefig((name if overwrite else (name+'_HD24.png')), dpi=184)
        else:
            fig.set_size_inches(27.8,15.4)
            if not legend:
                fig.subplots_adjust(left=0.04, right=0.98, top=0.96, bottom=0.04)
            else:
                fig.subplots_adjust(left=0.04, right=0.72, top=0.96, bottom=0.04)
            fig.savefig((name if overwrite else (name+'_4k32.png')), dpi=276)
    