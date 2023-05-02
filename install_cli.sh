#/bin/bash
initial_buri_version=v0.11.9

os=$(wc -l < $1)
arch=$(wc -l < $2)

if [[ -z "$os" ]]; then
	os=linux
    #os=darwin
fi
if [[ -z "$arch" ]]; then
    arch=amd64
    #arch=arm64
fi

# install initial version
curl https://mvnrepo.cantara.no/content/repositories/releases/no/cantara/gotools/buri/${initial_buri_version}/buri-${initial_buri_version}-${os}-${arch} -o /usr/local/bin/buri-${initial_buri_version}-${os}-${arch}
chmod +x /usr/local/bin/buri-${initial_buri_version}-${os}-${arch}
ln -s buri-${initial_buri_version}-${os}-${arch} /usr/local/bin/buri

# update buri to latest
buri install go -a buri -g no/cantara/gotools

# install nerthus-cli
buri install go -a nerthus-cli -g no/cantara/gotools

