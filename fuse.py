import pandas as pd
import numpy as np

data1 = pd.read_csv("seq_fetch_times.csv", index_col=0)
data2 = pd.read_csv("./fetch_benchmarks/noRepSel-seq_fetch_times_part1.csv", index_col=0)
data = (data1 + data2)/2
csv_file = "noRepSel-seq_fetch_times.csv"
data.to_csv(csv_file)