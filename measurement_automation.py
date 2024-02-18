from measure_web_fetch import fetch_and_time

def main():
    website="http://www.scion-sample.org"
    urls=[website+"/"+suffix for suffix in ["",
          "favicon.ico"]]+\
         [website+":8181/"+suffix for suffix in [
          "hello-world","form","sample-json",
          "sample-text","sample-image",
          "sample-gif","sample-audio",
          "sample-video"]]+\
         [website+":8899/"+suffix for suffix in [
          "SCION_Lec.m3u8","SCION_Lec_100.m4s"]]
    for url in urls:
        successfull_fetches, fetch_time = 0,0.0
        for _ in range(1000):
            response, time_taken = fetch_and_time(url)
            if response.ok: 
                successfull_fetches += 1
                fetch_time += time_taken
        if successfull_fetches>0:
            print("It took {:.9f} seconds to fetch {:s}".format((fetch_time/successfull_fetches),url))
            print(f"Was fetched {successfull_fetches} succesfully!")
        else:
            print(f"{url} was not fetched succesfully!")

if __name__ == "__main__":
    main()