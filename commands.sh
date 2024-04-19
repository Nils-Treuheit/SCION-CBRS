# install
echo "<scion-address> www.scion-sample.org" >> /etc/hosts
cd ~
sudo apt-get install apt-transport-https ca-certificates
echo "deb [trusted=yes] https://packages.netsec.inf.ethz.ch/debian all main" | sudo tee /etc/apt/sources.list.d/scionlab.list
sudo apt-get update
sudo apt-get install scionlab
sudo scionlab-config --host-id=<AS_ID>
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt update
sudo apt install golang-1.20
cd /bin
sudo ln -s go /usr/lib/go-1.20/bin/go
sudo ln -s gofmt /usr/lib/go-1.20/bin/gofmt
cd ~
sudo apt-get install -y libpam0g-dev
git clone https://github.com/netsec-ethz/scion-apps
cd scion-apps && make setup_lint && make -j && make install
cd ~
git clone https://github.com/Nils-Treuheit/SCION-CDN


# runtime
cd ~/scion-apps && ./bin/scion-skip
ssh -t <user>@<ip> "cd ~/SCION-CDN && go run scion_server.go selectors.go servers.go <mode>"
cd ~/SCION-CDN && python measurement_automation.py <reps> <mode> <debug>
