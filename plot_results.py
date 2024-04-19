import matplotlib.pyplot as plt
from sys import argv
import pandas as pd
import numpy as np


filename = "fetch_times.csv"
if len(argv)>1: filename = argv[1]
data = pd.read_csv(filename, index_col=0)
data.index = range(len(data.index))

for col in data.columns:
    fetches = len(data[col])
    successfull = fetches-data[col].isna().sum()
    fetch_times = data[col][~data[col].isna()] 
    print("Fetching",col,"results:")
    print("=> {:.4f}% of all URL fetches have been successfull.".format(successfull/fetches * 100))
    print("=> The average fetch time was {:.9f} seconds".format(np.mean(fetch_times)))
    print("=> The median fetch time was {:.9f} seconds".format(np.median(fetch_times)))
    print("=> The standard deviation was {:.6f}".format(np.std(fetch_times)))
    print("=> The fastest fetch took {:.9f}".format(np.min(fetch_times)))
    print("=> The slowest fetch took {:.9f}".format(np.max(fetch_times)))
    plt.plot(fetch_times,label=col.split("/")[-1])

plt.suptitle(("Parallel " if "par" in filename else "Sequential ")+"Fetching Stats")
plt.title(f"URL Fetch Times from {data.columns[0]}")
plt.xlabel("Fetch ID")
plt.ylabel("Time (sec)")
plt.yscale('log')
plt.legend(loc='upper left', bbox_to_anchor=(1.05, 1))
plt.tight_layout()
plt.savefig(filename.split("_")[0]+"_fetch_stats.png")
plt.show()