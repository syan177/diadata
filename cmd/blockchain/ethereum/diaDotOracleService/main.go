package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/diadata-org/diadata/pkg/dia/scraper/blockchain-scrapers/blockchains/ethereum/diaOracleService"
	models "github.com/diadata-org/diadata/pkg/model"
	"github.com/diadata-org/diadata/pkg/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	key := utils.Getenv("PRIVATE_KEY", "")
	key_password := utils.Getenv("PRIVATE_KEY_PASSWORD", "")
	deployedContract := utils.Getenv("DEPLOYED_CONTRACT", "")
	blockchainNode := utils.Getenv("BLOCKCHAIN_NODE", "")
	sleepSeconds, err := strconv.Atoi(utils.Getenv("SLEEP_SECONDS", "120"))
	if err != nil {
		log.Fatalf("Failed to parse sleepSeconds: %v")
	}
	frequencySeconds, err := strconv.Atoi(utils.Getenv("FREQUENCY_SECONDS", "120"))
	if err != nil {
		log.Fatalf("Failed to parse frequencySeconds: %v")
	}
	chainId, err := strconv.ParseInt(utils.Getenv("CHAIN_ID", "1"), 10, 64)
	if err != nil {
		log.Fatalf("Failed to parse chainId: %v")
	}

	addresses := []string{
		"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", //USDC
		"0xdAC17F958D2ee523a2206206994597C13D831ec7", //USDT
		"0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", //BUSD
		"0x6B175474E89094C44Da98b954EedeAC495271d0F", //DAI
		"0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", //WBTC
		"0x0000000000000000000000000000000000000000", //ETH
		"0x0000000000000000000000000000000000000000", //BNB
		"0x6bD193Ee6D2104F14F94E2cA6efefae561A4334B", //SOLAR
		"0x98878B06940aE243284CA214f92Bb71a2b032B8A", //WMOVR
	}

	blockchains := []string{
		"Ethereum", //USDC
		"Ethereum", //USDT
		"BinanceSmartChain", //BUSD
		"Ethereum", //DAI
		"Ethereum", //WBTC
		"Ethereum", //ETH
		"BinanceSmartChain", //BNB
		"Moonriver", //SOLAR
		"Moonriver", //WMOVR
	}

	/*
	 * Setup connection to contract, deploy if necessary
	 */

	conn, err := ethclient.Dial(blockchainNode)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	auth, err := bind.NewTransactorWithChainID(strings.NewReader(key), key_password, big.NewInt(chainId))
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}

	var contract *diaOracleService.DIAOracle
	err = deployOrBindContract(deployedContract, conn, auth, &contract)
	if err != nil {
		log.Fatalf("Failed to Deploy or Bind contract: %v", err)
	}

	/*
	 * Update Oracle periodically with coins
	 */
	ticker := time.NewTicker(time.Duration(frequencySeconds) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				for i, address := range addresses {
					blockchain := blockchains[i]
					err = periodicOracleUpdateHelper(auth, contract, conn, blockchain, address)
					if err != nil {
						log.Println(err)
					}
					time.Sleep(time.Duration(sleepSeconds) * time.Second)
				}
			}
		}
	}()
	select {}
}

func periodicOracleUpdateHelper(auth *bind.TransactOpts, contract *diaOracleService.DIAOracle, conn *ethclient.Client, blockchain string, address string) error {
	// Get quotation for token and update Oracle
	rawQ, err := getAssetQuotationFromDia(blockchain, address)
	if err != nil {
		log.Fatalf("Failed to retrieve %s quotation data on chain %s from DIA: %v", address, blockchain, err)
		return err
	}
	rawQ.Name = rawQ.Symbol

	err = updateQuotation(rawQ, auth, contract, conn)
	if err != nil {
		log.Fatalf("Failed to update DIA Oracle: %v", err)
		return err
	}
	return nil
}

func deployOrBindContract(deployedContract string, conn *ethclient.Client, auth *bind.TransactOpts, contract **diaOracleService.DIAOracle) error {
	var err error
	if deployedContract != "" {
		*contract, err = diaOracleService.NewDIAOracle(common.HexToAddress(deployedContract), conn)
		if err != nil {
			return err
		}
	} else {
		// deploy contract
		var addr common.Address
		var tx *types.Transaction
		addr, tx, *contract, err = diaOracleService.DeployDIAOracle(auth, conn)
		if err != nil {
			log.Fatalf("could not deploy contract: %v", err)
			return err
		}
		log.Printf("Contract pending deploy: 0x%x\n", addr)
		log.Printf("Transaction waiting to be mined: 0x%x\n\n", tx.Hash())
		time.Sleep(180000 * time.Millisecond)
	}
	return nil
}

func updateQuotation(quotation *models.Quotation, auth *bind.TransactOpts, contract *diaOracleService.DIAOracle, conn *ethclient.Client) error {
	symbol := quotation.Symbol + "/USD"
	price := quotation.Price
	timestamp := time.Now().Unix()
	err := updateOracle(conn, contract, auth, symbol, int64(price*100000000), timestamp)
	if err != nil {
		log.Fatalf("Failed to update Oracle: %v", err)
		return err
	}

	return nil
}

func updateOracle(
	client *ethclient.Client,
	contract *diaOracleService.DIAOracle,
	auth *bind.TransactOpts,
	key string,
	value int64,
	timestamp int64) error {

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Get 110% of the gas price
	fmt.Println(gasPrice)
	fGas := new(big.Float).SetInt(gasPrice)
	fGas.Mul(fGas, big.NewFloat(1.1))
	gasPrice, _ = fGas.Int(nil)
	fmt.Println(gasPrice)
	// Write values to smart contract
	tx, err := contract.SetValue(&bind.TransactOpts{
		From:     auth.From,
		Signer:   auth.Signer,
		GasLimit: 1000725,
		GasPrice: gasPrice,
	}, key, big.NewInt(value), big.NewInt(timestamp))
	if err != nil {
		return err
	}
	fmt.Println(tx.GasPrice())
	log.Printf("key: %s\n", key)
	log.Printf("Tx To: %s\n", tx.To().String())
	log.Printf("Tx Hash: 0x%x\n", tx.Hash())
	return nil
}

func getAssetQuotationFromDia(blockchain, address string) (*models.Quotation, error) {
	response, err := http.Get("https://rest.diadata.org/v1/assetQuotation/" + blockchain + "/" + address)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	if 200 != response.StatusCode {
		return nil, fmt.Errorf("Error on dia api with return code %d", response.StatusCode)
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var quotation models.Quotation
	err = quotation.UnmarshalBinary(contents)
	if err != nil {
		return nil, err
	}
	return &quotation, nil
}
