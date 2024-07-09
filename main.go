package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

func main() {
	//Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error - could not load .env file")
	}

	privateKey, ok := os.LookupEnv("PRIVATE_KEY")
	if !ok {
		log.Printf("error - PRIVATE_KEY environment variable not set")
	}

	chainId, ok := os.LookupEnv("CHAIN_ID")
	if !ok {
		log.Printf("error - PRIVATE_KEY environment variable not set")
	}

	explorerURL, ok := os.LookupEnv("EXPLORER_URL")
	if !ok {
		fmt.Printf("error - EXPLORER_URL variable not set in .env\n")
	}

	rpcProvider, ok := os.LookupEnv("RPC_PROVIDER")
	if !ok {
		fmt.Printf("error - RPC_PROVIDER variable not set in .env\n")
	}

	client, err := ethclient.Dial(rpcProvider)
	if err != nil {
		log.Fatal("error - Could not connect to RPC provider", err)
	}

	uint256Value, ok := os.LookupEnv("UINT256_VALUE")
	if !ok {
		log.Printf("error - UINT256_VALUE environment variable not set")
	}

	//Parse env variables
	privateKeyParsed, err := crypto.HexToECDSA(strings.TrimSpace(privateKey))
	if err != nil {
		log.Fatal("error - parsing privateKey failed", err)
	}

	chainIdParsed := new(big.Int)
	_, success := chainIdParsed.SetString(chainId, 10)
	if !success {
		log.Fatal("error - parsing chainId failed")
		return
	}

	uint256ValueParsed := new(big.Int)
	_, success = uint256ValueParsed.SetString(uint256Value, 10)
	if !success {
		log.Fatal("error - parsing uint256Value failed")
		return
	}

	//Read contract
	contractFiles, err := os.ReadDir("./contracts")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range contractFiles {
		{
			//Compile contract
			if filepath.Ext(file.Name()) == ".sol" {
				contractPath := filepath.Join("contracts", file.Name())

				var cmd = exec.Command("solc", "--bin", "--abi", "--optimize", "--output-dir", "compiled-contracts", "--evm-version", "constantinople", "--overwrite", contractPath)

				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					log.Fatal(err)
				}
			}

			binDir, err := filepath.Abs("./compiled-contracts")
			if err != nil {
				log.Fatal(err)
			}

			name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			binFilename := fmt.Sprintf("%s.bin", name)
			abiFilename := fmt.Sprintf("%s.abi", name)
			binPath := filepath.Join(binDir, binFilename)
			abiPath := filepath.Join(binDir, abiFilename)

			bytecodeBytes, err := os.ReadFile(binPath)
			if err != nil {
				log.Fatal(err)
			}
			bytecodeStr := string(bytecodeBytes)
			constructorBytes, err := hex.DecodeString(bytecodeStr[:len(bytecodeStr)-68])
			if err != nil {
				log.Fatal(err)
			}

			abiBytes, err := os.ReadFile(abiPath)
			if err != nil {
				log.Fatal(err)
			}

			contractABI, err := abi.JSON(bytes.NewReader(abiBytes))
			if err != nil {
				log.Fatal(err)
			}

			//Set gas price
			gasPrice, err := client.SuggestGasPrice(context.Background())
			if err != nil {
				log.Fatal(err)
			}

			estimateGas, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
				From: crypto.PubkeyToAddress(privateKeyParsed.PublicKey),
				To:   nil,
				Data: constructorBytes,
			})
			if err != nil {
				log.Fatal(err)
			}

			if err != nil {
				fmt.Printf("estimate gas overflow uint64\n")
				log.Fatal(err)
			}

			//Sign transaction
			auth, err := bind.NewKeyedTransactorWithChainID(privateKeyParsed, chainIdParsed)
			if err != nil {
				log.Fatal(err)
			}
			gasLimit := estimateGas
			auth.GasPrice = gasPrice
			auth.GasLimit = gasLimit + uint64(100000)

			fmt.Printf("start - deploying contract %s\n", file.Name())

			//Deploy the contract
			address, tx, _, err := bind.DeployContract(auth, contractABI, constructorBytes, client)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("pending - contract %s waiting for transaction: %s\n", file.Name(), explorerURL+"/tx/"+tx.Hash().Hex())
			receipt, err := bind.WaitMined(context.Background(), client, tx)
			if err != nil {
				log.Fatal(err)
			}
			if receipt.Status != types.ReceiptStatusSuccessful {
				log.Fatalf("error - contract %s deployment failed", file.Name())
			}

			fmt.Printf("success - contract %s deployed: %s\n", file.Name(), explorerURL+"/address/"+address.Hex())

			//Set value via setUint256
			setUint256 := func(client *ethclient.Client, contractAddress common.Address, auth *bind.TransactOpts, value *big.Int) (*types.Transaction, error) {

				instance := bind.NewBoundContract(contractAddress, contractABI, client, client, client)

				tx, err := instance.Transact(auth, "setUint256", value)
				if err != nil {
					return nil, fmt.Errorf("failed to call setUint256: %s", err)
				}

				return tx, nil
			}

			tx, err = setUint256(client, address, auth, uint256ValueParsed)
			if err != nil {
				log.Fatalf("failed to call setUint256: %s", err)
			}

			fmt.Printf("pending - setUint256 waiting for transaction: %s\n", explorerURL+"/tx/"+tx.Hash().Hex())
			receipt, err = bind.WaitMined(context.Background(), client, tx)
			if err != nil {
				log.Fatal(err)
			}
			if receipt.Status != types.ReceiptStatusSuccessful {
				log.Fatalf("error - setUint256 transaction failed")
			}

			fmt.Println("success - setUint256 transaction completed")

			//Get value via getUint256
			getUint256 := func(client *ethclient.Client, contractAddress common.Address) (*big.Int, error) {
				instance := bind.NewBoundContract(contractAddress, contractABI, client, client, client)

				var rawResult []interface{}
				callOpts := &bind.CallOpts{}
				err := instance.Call(callOpts, &rawResult, "getUint256")
				if err != nil {
					return nil, fmt.Errorf("failed to call getUint256: %s", err)
				}

				if len(rawResult) == 0 {
					return nil, fmt.Errorf("empty result returned from getUint256")
				}

				result, ok := rawResult[0].(*big.Int)
				if !ok {
					return nil, fmt.Errorf("unexpected type returned from getUint256: %s", rawResult[0])
				}

				return result, nil
			}

			currentValue, err := getUint256(client, address)
			if err != nil {
				log.Fatalf("failed to call getUint256: %v", err)
			}

			fmt.Printf("success - getUint256 current value is: %s\n", currentValue.String())

			//Write the output data to a JSON
			data := map[string]string{"getUint256Value": currentValue.String(), "deployerAddress": auth.From.Hex(), "contractAddress": address.Hex()}

			file, err := os.Create("output.json")
			if err != nil {
				fmt.Println("error - creating file failed", err)
				return
			}
			defer file.Close()

			jsonData, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				fmt.Println("error -  encoding JSON failed", err)
				return
			}

			_, err = file.Write(jsonData)
			if err != nil {
				fmt.Println("error - writing to file failed:", err)
				return
			}
		}
	}
}
