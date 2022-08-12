module github.com/polynetwork/poly-relayer

go 1.15

require (
	github.com/btcsuite/btcd v0.22.0-beta
	github.com/ethereum/go-ethereum v1.9.25
	github.com/go-redis/redis/v8 v8.11.3
	github.com/joeqian10/neo-gogogo v1.4.0
	github.com/ontio/ontology v1.14.0-beta.0.20210818114002-fedaf66010a7
	github.com/ontio/ontology-crypto v1.2.1
	github.com/polynetwork/bridge-common v0.0.32-plt
	github.com/polynetwork/eth-contracts v0.0.0-20200814062128-70f58e22b014
	github.com/polynetwork/poly v1.3.1
	github.com/polynetwork/poly-go-sdk v0.0.0-20210114035303-84e1615f4ad4
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1
)

replace github.com/ethereum/go-ethereum => github.com/dylenfu/palette v0.0.0-20210817120114-6e0ae4f73447
