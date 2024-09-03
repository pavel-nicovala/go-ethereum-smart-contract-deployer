# go-ethereum-smart-contract-deployer
Go poc for a simple smart contract deployer 

## Dependencies:

```
- go
- solc
```

### Usage

_Config_

A valid wallet private key needs to be sourced for PRIVATE_KEY in .env, this poc is configured for holesky network but can be changed to any other network if valid .env variables are sourced.         

_Run_
```sh            
go run main.go
```


### Description:

Once run script will compile the .sol contract from contracts folder, 
will create .abi and .bin under compiled-contracts folder, 
start deploying using the env variables configured in .env, 
query the contract once deployed and write "contractAddress", "deployerAddress" 
and "getUint256Value" from the contract in an output.josn file.
