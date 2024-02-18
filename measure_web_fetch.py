import requests
from bs4 import BeautifulSoup
import time
import sys

PROXIES = {
    'http': 'http://127.0.0.1:8888/skip.pac',
    'https': 'https://127.0.0.1:8888/skip.pac',
}


def fetch_and_time(url):
    global PROXIES
    start_time = time.time()
    response = requests.get(url,proxies=PROXIES)
    elapsed_time = time.time() - start_time
    return response, elapsed_time

def main(url):
    # Fetch the main website
    response, time_taken = fetch_and_time(url)
    print(f"Time taken to fetch {url}: {time_taken} seconds")

    if response.status_code == 200:
        # Parse the HTML content
        soup = BeautifulSoup(response.content, 'html.parser')
        
        # Find all links
        href_links = soup.find_all('a', href=True)+\
                     soup.find_all('link', href=True)
        src_links = soup.find_all('img', src=True)+\
                    soup.find_all('source', src=True)
        
        # Download each link and measure time
        for link in href_links:
            link_url = link['href']
            if not link_url.startswith('http'):
                link_url = url + link_url

            try:
                _, link_time = fetch_and_time(link_url)
                print(f"Time taken to fetch {link_url}: {link_time} seconds")
            except requests.RequestException as e:
                print(f"Error fetching {link_url}: {e}")

        for link in src_links:
            link_url = link['src']
            if not link_url.startswith('http'):
                link_url = url + link_url

            try:
                _, link_time = fetch_and_time(link_url)
                print(f"Time taken to fetch {link_url}: {link_time} seconds")
            except requests.RequestException as e:
                print(f"Error fetching {link_url}: {e}")

if __name__ == "__main__":
    if len(sys.argv) > 1:
        main(sys.argv[1])
    else:
        print("Please provide a URL as an argument.")


