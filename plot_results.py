import matplotlib.pyplot as plt
import pandas as pd

filename = "fetch_times.csv"
data = pd.read_csv(filename)

for col in data.columns:
    fetches = len(data[col])
    successful = fetches-data[col].isna().sum()
    fetch_times = data[col][~data[col].isna()] 
    plt.plot(fetch_times,label=col)

plt.title("URL Fetch Times")
plt.xlabel("Fetch ID")
plt.ylabel("Time (sec)")
plt.legend()
plt.show()