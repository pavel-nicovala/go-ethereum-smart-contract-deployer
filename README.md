# go-ethereum-smart-contract-deployer
Go poc for a simple smart contract deployer 

## Dependencies:

```
- go
- solc
```

### Usage

_Config_

```sh
A valid wallet private key needs to be sourced for PRIVATE_KEY in .env        
```

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
