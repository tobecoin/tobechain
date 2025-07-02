<h1 align="center">Tobe Chain</h1>
<p align="center">The Efficient Blockchain Powered By Proof of Intelligence Consensus.</p>
<div align="center">

[Tobe Chain](https://github.com/tobecoin/tobechain/actions)

</div>

Golang execution layer implementation of the Tobechain protocol.


More details can be found at our [TobeChain](https://tobechain.net/)

Read more about us on:
- Our documentation portal: [TobeChain Documentation](https://docs.tobescan.com/docs/about/welcome)
- Our blockchain explorer: [TobeScan](https://www.tobescan.com/)


## How To Build

TobeChain supports binaries build. Building `tobe` requires both a Go (version < go1.23.0) and a C compiler on all platforms (Linux, Window, MacOS).

Due to some changes in Go-lang, TobeChain need to be build with Go `1.20 - 1.23.` You can install them using your favourite package manager.

```bash
go install golang.org/dl/go1.23.0@latest
go1.23.0 download
```

#### Build `tobe`

Clone this repository and change working directory to where you clone it, then run the following commands:

```shell
make tobe
```

or, to build the full suite of utilities:

```shell
make all
```

## Running `tobe`

Going through all the possible command line flags is out of scope here, but we've enumerated a few common parameter combos to get you up to speed quickly on how you can run your own `tobe` instance.

### Initialize accounts for the node keystore

TobeChain requires an account when running the node, even it's a full node. If you already had an existing account, import it. Otherwise, please initialize new accounts.

Initialize new account

```bash
mkdir tobechain/ && cd tobechain/
cp ./build/bin/tobe /tobechain/
./tobe account new --datadir /tobechain
```
### Run a full node on Tobechain

Initial `genesis.json` file to init Tobechain

```bash
./tobe --datadir /tobechain init genesis.json
```

Export `keystore` file

```bash
export KEYSTORE_PATH=/path/to/your/keystore_file
Ex: export KEYSTORE_PATH=/tobechain/keystore/UTC--...
```

To run full node with default settings, simply run this command.

```bash
./tobe --datadir /tobechain --unlock 0
```

The following also run full node, with more customization
```bash
./tobe --datadir /tobechain \
--networkid 7090 \
--unlock "Your Wallet Address" \
--mine --miner.etherbase "Your Wallet Address" \ 
--http --http.api "eth,net,web3,miner" \
--allow-insecure-unlock \
--nodiscover \
--http.port 8545 
--port 30303
```

Brief explainations on the used flags:

```text
--datadir: path to your data directory created above.
--keystore: path to your account's keystore created above.
--password: your account's password.
--identity: your full node's name.
--networkid: our network ID.
--gasprice: Minimal gas price to accept for mining a transaction.
--rpc, --rpcaddr, --rpcport, --rpcvhosts, --rpccorsdomain: configure HTTP-RPC.
--ws, --wsaddr, --wsport, --wsorigins: configure Websocket.
--mine: your full-node wants to register to be a candidate for masternode selection.
--bootnodes: list of enodes of other peers that your full-node will try to connect at startup
--port: your full-node's listening port (default to 30303)
--nat NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>) to let other peer connect to your node easier
--synmode: blockchain sync mode ("fast", "full", or "light".)
--gcmode: blockchain garbage collection mode ("full", "archive")
--store-reward: store reward report. must be used in conjuction with --gcmode archive for archive node
--ethstats: send data to stats website
--verbosity: log level from 1 to 5. Here we're using 4 for debug messages
```


## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) are licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.
