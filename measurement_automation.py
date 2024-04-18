''' Usage Instructions 
- make sure the remote scion servers are running and reachable
- check if scion skip is running
- check if urls in main-method and proxies are correct  
-> run program in terminal like this: 

   python measurement_automation.py 100 s 1

 first argument => how many fetch runs to approximate stable average
 second argument => flag for runtime mode 0 = Sequential, 1 = Parallel
 third argument => flag to enable detailed stats output (mainly for debug) 
'''
from measure_web_fetch import fetch_and_time
import concurrent.futures as thread_lib
from os import system
from time import time
from sys import argv
import pandas as pd
import numpy as np

PROXIES = {
    'http': 'http://127.0.0.1:8888/skip.pac',
    'https': 'https://127.0.0.1:8888/skip.pac',
}
SEQUENTIAL = False
PARALLEL = True


def fetch_urls_in_parallel(urls, runs, prox=PROXIES):
    data = pd.DataFrame(columns=urls, index=[f"fetch_{id}" for id in range(runs)])

    def fetch_single_url(url,prox):
        col = dict()
        for id in range(runs):
            response, time_taken = fetch_and_time(url,proxies=prox)
            if response.ok: col[f"fetch_{id}"] = time_taken
            else: col[f"fetch_{id}"] = float('NaN')
        return col

    with thread_lib.ThreadPoolExecutor() as executor:
        futures = {executor.submit(fetch_single_url, url, prox): url for url in urls}
        for future in thread_lib.as_completed(futures):
            url = futures[future]
            data[url] = future.result()
    return data


def fetch_urls_sequential(urls, runs, debug, prox=PROXIES):
    data = pd.DataFrame(columns=urls,index=[f"fetch_{id}" for id in range(runs)])
    for url in urls:
        if debug: successfull_fetches, fetch_times = 0,list()
        col = dict()
        for id in range(runs):
            response, time_taken = fetch_and_time(url, prox)
            if response.ok: 
                col[f"fetch_{id}"] = time_taken
                if debug:
                    successfull_fetches += 1
                    fetch_times.append(time_taken)
            else: col[f"fetch_{id}"] = float('NaN')
        data[url] = col
        if debug:
            if successfull_fetches>0:
                print("It took {:.9f} seconds to fetch benchmark {:s}".format(np.sum(fetch_times),url))
                print("-> The average fetch time was {:.9f} seconds".format(np.mean(fetch_times)))
                print("-> The median fetch time was {:.9f} seconds".format(np.median(fetch_times)))
                print("-> The standard deviation was {:.6f}".format(np.std(fetch_times)))
                print("-> The fastest fetch took {:.9f}".format(np.min(fetch_times)))
                print("-> The slowest fetch took {:.9f}".format(np.max(fetch_times)))
                print(f"-> Was fetched {successfull_fetches} succesfully!\n")
            else:
                print(f"{url} was not fetched succesfully!\n")
    return data


def main(runs, fetchmode, debug):
    website="http://www.scion-sample.org"
    urls=[website+"/"+suffix for suffix in ["",
          "favicon.ico","hello-world",
          "sample-json","sample-text"]]+\
         [website+":8899/"+suffix for suffix in [
          "SCION_Lec.m3u8","SCION_Lec_100.m4s"]]+\
         [website+":8181/"+suffix for suffix in [
          "sample-image","sample-gif",
          "sample-audio","sample-video"]]
    start = time()
    data = fetch_urls_in_parallel(urls, runs) if fetchmode==PARALLEL else fetch_urls_sequential(urls, runs, debug)
    end = time()
    print("The whole",("Parallel" if fetchmode==PARALLEL else "Sequential"),f"Fetching Benchmark took {end-start} seconds.")
    csv_file = ("par_" if fetchmode==PARALLEL else "seq_")+"fetch_times.csv"
    data.to_csv(csv_file)  
    system(f"python plot_results.py {csv_file}")
    

if __name__ == "__main__":
    runs = 100
    fetchmode=SEQUENTIAL
    debug=True

    if len(argv)>1:
        runs = int(argv[1])
    if len(argv)>2 and ('p' in argv[2].lower() or argv[2]=='1'):
        fetchmode = PARALLEL
    if fetchmode==PARALLEL or (len(argv)>3 and int(argv[3])==0):
        debug = False

    print("Run Fetch Program "+("in parallel " if fetchmode==PARALLEL else "sequentially ")+("with" if debug else "without")+" debug output")
    main(runs, fetchmode, debug)