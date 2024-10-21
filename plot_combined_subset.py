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

#COLORS = ["lightskyblue", 
#          "palegreen",  
#          "aqua", 
#          "lime",
#          "navy",
#          "seagreen", 
#          "royalblue", 
#          "yellowgreen"]

COLORS = ["fuchsia","pink","plum","deeppink"]
          

FILES_SUBSET = ["noRepSel_par.csv",
                "latrs_100it_5p_par.csv",
                "latrs_100it_20p_par.csv",
                "zippo_100it_4o9p_par.csv"] 
SUBSET_IDS = ["default","latrs(5p)","latrs(20p)","zippo"]
SELECTED_COLUMNS = [#"http://www.scion-sample.org:8899/SCION_Lec_100.m4s",
                    "http://www.scion-sample.org:8181/sample-video"]
NAME = "Parallel_Video_Fetching_Results"

legend = True if len(argv)>4 and int(argv[4])>0 else False
overwrite = True if len(argv)>3 and int(argv[3])>0 else False
res_format = ResFromat(int(argv[2])) if len(argv)>2 else ResFromat.UHD32
if len(argv)>1: 
    directory = argv[1]
    prefix = abspath(directory)
    
    data = pd.DataFrame()
    for idx,filename in enumerate(FILES_SUBSET):
        data_piece = pd.read_csv(prefix+"/"+filename, index_col=0)
        data_piece.index = range(len(data_piece.index))
        for col in SELECTED_COLUMNS: data[SUBSET_IDS[idx]+"/"+col.split("/")[-1]] = data_piece[col]
    
    fig = plt.figure(0,figsize=(20, 11)) 
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
        ax.plot(fetch_times,c=COLORS[id],label=col)

    #fig.suptitle(((("           " if not legend else "")+"Parallel ") if "par" in NAME.lower() else (("         " if not legend else "")+"Sequential "))+"Fetching Stats")
    ax.set_title(("Parallel " if "par" in NAME.lower() else "Sequential ")+"Fetching Stats")
    ax.set_xlabel("Fetch ID")
    ax.set_ylabel("Time (sec)")
    #ax.set_yscale('log')
    if legend: ax.legend(loc='upper left', bbox_to_anchor=(1.02, 1))
    fig.tight_layout()
    
    if res_format == ResFromat.A4_half:
        fig.set_size_inches(4,3)
        if not legend:
            fig.subplots_adjust(left=0.15, right=0.97, top=0.89, bottom=0.16)
        else:
            fig.subplots_adjust(left=0.15, right=0.62, top=0.88, bottom=0.16)
        fig.savefig((NAME if overwrite else (NAME+'_half_A4.png')), dpi=200)
    elif res_format == ResFromat.A4_full:
        fig.set_size_inches(8,4)
        if not legend:
            fig.subplots_adjust(left=0.07, right=0.98, top=0.90, bottom=0.13)
        else:
            fig.subplots_adjust(left=0.07, right=0.64, top=0.90, bottom=0.13)
        fig.savefig((NAME if overwrite else (NAME+'_full_A4.png')), dpi=400)
    elif res_format == ResFromat.HD24:
        fig.set_size_inches(20.3,11.5)
        if not legend:
            fig.subplots_adjust(left=0.06, right=0.98, top=0.96, bottom=0.06)
        else:
            fig.subplots_adjust(left=0.06, right=0.64, top=0.96, bottom=0.06)
        fig.savefig((NAME if overwrite else (NAME+'_HD24.png')), dpi=368)
    else:
        fig.set_size_inches(27.8,15.4)
        if not legend:
            fig.subplots_adjust(left=0.04, right=0.98, top=0.98, bottom=0.04)
        else:
            fig.subplots_adjust(left=0.04, right=0.64, top=0.98, bottom=0.04)
        fig.savefig((NAME if overwrite else (NAME+'_4k32.png')), dpi=552)
    