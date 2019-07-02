# go-seele
[![Build Status](https://travis-ci.org/seeleteam/go-seele.svg?branch=master)](https://travis-ci.org/seeleteam/go-seele)

|        Features        |      Description                                                                               |
|:-----------------------|------------------------------------------------------------------------------------------------|
| **Sharding**           | 4 shards, transactions within the same shard and between different shards are supported<br/> higher transaction fee for cross-shard transaction                                  |
| **Smart Contracts**    | smart contracts are supported within the same shard                                          |
| **Seele Wallet**       | easy-to-use wallet                                                                             |
| **High TPS**           | same shard TPS: 500/shard, cross shard TPS: 12/shard                                           |
| **Auditable Supply**   | total supply: 1,000,000,000 SEELEs, 300,000,000 SEELEs for mining                              |
| **Consensus Algorithm**| Seele POW: memory heavy proof of work algorithm                                                |
| **Mining Reward**      | 3150000 blocks/era and era reward follows [6, 4, 3, 2.5, 2, 2, 1.5, 1.5] order until reaches the last reward of 1.5 SEELEs |
| **Transaction Fee**    | self-customized transaction fee, higher fee for cross-shard transaction                        |
| **Block**              | 100 KB block size, 10 seconds block time, ~6000 transactions per block                         |


The official Golang implementation of Seele. Seele blockchain is based on the Neural Consensus protocol, which consists of a mainchain and subchains to form a heterogeneous forest multi-chain ecosystem when fully developed. [https://seele.pro](https://seele.pro)

The current mainnet release: Seele mainchain is powered by a new anti-ASIC consensus PoW algorithm, which requires scientific calculation related to matrix. [MPOW PAPER](https://arxiv.org/abs/1905.04565) The mainchain has four shards. It can perform transactions within a shard or crossing shards. However, smart contracts currently can be only executed within the same shard. Seele subchains are under development. The so called EDA consensus algorithm [EDA PAPER](http://seele.hk.ufileos.com/Seele_Yellow_Paper_EDA_A_Parallel_Data_Sorting_Mechanism_for_Distributed_Information_Processing_System_Pre-Release.pdf) from Seele will be utilized for the subchains.

# Downloading (without building)
If you want to directly run the node and use client without setting up the compiling enviroment and building the executable files, you can choose right version to download and run:

| Operation System |      Download Link     |
|---------|----------------------------------------------------------|
| Linux   | [https://github.com/seeleteam/go-seele/releases/tag/v1.0.1]|
| MacOs   | [https://github.com/seeleteam/go-seele/releases/tag/v1.0.1]|
| Windows | [https://github.com/seeleteam/go-seele/releases/tag/v1.0.1]|

# Or Downloading & Building the source

Building the Seele project requires both a Go (version 1.7 or later) compiler and a C compiler. You can install them using your favourite package manager. Once the dependencies are installed, run

- Building the Seele project requires both a Go (version 1.7 or later) compiler and a C compiler. Install Go v1.10 or higher, Git, and the C compiler.

- Clone the go-seele repository to the GOPATH directory:

```
go get -u -v github.com/seeleteam/go-seele/... 
```

- Once successfully cloned source code:

```
cd GOPATH/src/github.com/seeleteam/go-seele/
```

- Linux & Mac

```
make all
```

- Windows

```
buildall.bat
```

# Running Seele
A simple version Seele mining tutorial: English-[SeeleMiningTutorial](https://github.com/seeleteam/go-seele/releases/tag/v1.0.1-MiningTutorial_Eng), 中文-[Seele挖矿教程中文简版](https://github.com/seeleteam/go-seele/releases/tag/v1.0.1-%E4%B8%AD%E6%96%87%E7%AE%80%E7%89%88%E6%8C%96%E7%9F%BF%E6%95%99%E7%A8%8B).

For running a node, please refer to [Get Started](https://seeleteam.github.io/seele-doc/docs/Getting-Started-With-Seele.html).
For more usage details and deeper explanations, please consult the [Seele Wiki](https://github.com/seeleteam/go-seele/wiki).

# Contribution

Thank you for considering helping out with our source code. We appreciate any contributions, even the smallest fixes.

Here are some guidelines before you start:
* Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
* Pull requests need to be based on and opened against the `master` branch.
* We use reviewable.io as our review tool for any pull request. Please submit and follow up on your comments in this tool. After you submit a PR, there will be a `Reviewable` button in your PR. Click this button, it will take you to the review page (it may ask you to login).
* If you have any questions, feel free to join [chat room](https://gitter.im/seeleteamchat/dev) to communicate with our core team.

# Resources

* [Seele Website](https://seele.pro/)
* [Dev Chat Room](https://gitter.im/seleeteam/dev)
* [Telegram Group](https://t.me/seeletech)
* [White Paper](https://s3.ap-northeast-2.amazonaws.com/wp.s3.seele.pro/Seele_White_Paper_English_v3.1.pdf)
* [Roadmap](https://seele.pro/)

# License

[go-seele/LICENSE](https://github.com/seeleteam/go-seele/blob/master/LICENSE)



