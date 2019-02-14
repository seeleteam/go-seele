# go-seele
[![Build Status](https://travis-ci.org/seeleteam/go-seele.svg?branch=master)](https://travis-ci.org/seeleteam/go-seele)



The official Golang implementation of Seele. Seele is powered by an up-scalable Neural Consensus protocol for high throughput concurrency among large scale heterogeneous nodes and is able to form a unique heterogeneous forest multi-chain ecosystem https://seele.pro

# Downloading & building the source

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



