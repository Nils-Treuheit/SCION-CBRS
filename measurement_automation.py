from measure_web_fetch import fetch_and_time
import numpy as np

def main():
    website="http://www.scion-sample.org"
    urls=[website+"/"+suffix for suffix in ["",
          "favicon.ico","hello-world",
          "sample-json","sample-text"]]+\
         [website+":8899/"+suffix for suffix in [
          "SCION_Lec.m3u8","SCION_Lec_100.m4s"]]+\
         [website+":8181/"+suffix for suffix in [
          "sample-image","sample-gif",
          "sample-audio","sample-video"]]
         
    for url in urls:
        successfull_fetches, fetch_times = 0,list()
        for _ in range(100):
            response, time_taken = fetch_and_time(url)
            if response.ok: 
                successfull_fetches += 1
                fetch_times.append(time_taken)
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

if __name__ == "__main__":
    main()