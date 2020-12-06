# consensus
Byzantine fault-tolerant consensus based on Raft cluster-RBFT

This project is part of meteorchain(Confidential, not open source yet). We proposed an algorithm that solves the efficiency and security problems of the blockchain consensus algorithm through network sharding. The algorithm is based on two basic algorithms, PBFT and Raft.

This project implements the network fragmentation method and consensus strategy of the proposed algorithm. The source code can refer to the source file, based on go 1.14. You can find Bolt database and xml configuration usage method in "github.com/boltdb/bolt" and "github.com/beevik/etree".

We organized and published the basic results of the test to this project. It is worth noting that the test uses a virtual machine local area network, the network environment is relatively good, the project only tested the consensus algorithm part, did not include other core parts of the blockchain, such as Merkle tree, MVC, the transaction data structure of the test is relatively simple. But all tests (RBFT, raft, PBFT) are based on the same environment, so it can still illustrate the problem.

## a brief usage description
* Gomod is required to build this project, run go mod vendor to download related dependencies.
* You can manually or automatically group nodes through the consistentHash module.see more details in "consistentHashgroup".
* run go build to build an app.
* the more usage canbe found in log files.
