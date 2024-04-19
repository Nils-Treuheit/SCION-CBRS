# SCION-CDN
Create a Benchmark to measure the impact of different communication paths on the content delivery of small SCION network using 2 ASes with several paths and hops between them. 

## Disclaimer
Content Files are purpose-oriented deviations of cited sources to benchmark different content network traffic. Content files will not be uploaded to the repository and might only be temporarily available under the sourced links as I do not plan to regular update them and have to refrain from redistributing content of other creators on the web. The website text is partially generated with ChatGPT and not fact checked as I just needed a larger chunk of text to download. <br><br>

## Results
You can find the results of my benchmarks and test in the [benchmark folder](./fetch_benchmarks)<br><br>

## Installation
This section describes how to install all necessary parts to get the benchmark up and running. You need a SCION installation with a SCION AS and a SCION apps installation, since you need to run the SCION proxy server skip and access a remote SCION AS. In case you struggle with the go installation view [go install instructions](https://go.dev/wiki/Ubuntu). Alternatively you can also find a compiled list of commands in [commands.sh](./commands.sh).<br>

**<i>local machine:</i>**<br>
-> Create an account and your own SCIONLab AS with the [SCIONLab Organisation](https://www.scionlab.org/login) <br>
-> Follow the [Installation and Configuration Guidelines](https://docs.scionlab.org/content/install/pkg.html) to setup scion on your machine <br>
-> Follow the build instructions of the [scion-apps repository](https://github.com/netsec-ethz/scion-apps)<br>
Then run the following lines and substitute <code>\<scion-address\></code> with the actual SCION address of your AS 
``` bash
git clone https://github.com/Nils-Treuheit/SCION-CDN
echo "<scion-address> www.scion-sample.org" >> /etc/hosts
```
**<i>remote machine:</i>**<br>
-> Create an account and your own SCIONLab AS with the [SCIONLab Organisation](https://www.scionlab.org/login) <br>
-> Follow the [Installation and Configuration Guidelines](https://docs.scionlab.org/content/install/pkg.html) to setup scion on your machine <br>
``` bash
git clone https://github.com/Nils-Treuheit/SCION-CDN
```
Make sure the local and remote machine are located in different networks and more importantly also be assigned to different SCION attachment points! 
<br><br>

## Runtime
To run the benchmark execute the following commands please make sure you substitute the elements in the <code><></code> brackets. You can find a compiled list of commands in [commands.sh](./commands.sh).
``` bash
cd ~/scion-apps && ./bin/scion-skip
ssh -t <user>@<ipv4> "cd ~/SCION-CDN && go run scion_server.go selectors.go servers.go <mode>"
cd ~/SCION-CDN && python measurement_automation.py <reps> <mode> <debug>
```