package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"reserved-gas-contract-script/utils"
	"sort"
	"strconv"
	"strings"
)

const (
	alchemyURL = "https://api.roninchain.com/rpc"
	EpochV2    = 200

	roninValidatorSet = "0x617c5d73662282ea7ffd231e020eca6d2b0d552f"
	slashIndicator    = "0xebfff2b32fa0df9c5c8c5d5aaa7e8b51d5207ba3"
	stakingContract   = "0x545edb750eb8769c868429be9586f5857a768758"
	profileContract   = "0x840ebf1ca767cb690029e91856a357a43b85d035"
	finalityTracking  = "0xa30b2932cd8b8a89e34551cdfa13810af38da576"
)

type BlockInfo struct {
	GasUsed      string        `json:"gasUsed"`
	Transactions []Transaction `json:"transactions"`
	// include other block fields you might need
}

type Response struct {
	Jsonrpc string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Result  BlockInfo `json:"result"`
}

type Response2 struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  Transaction `json:"result"`
}

type Transaction struct {
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Gas         string `json:"gas"`
	GasPrice    string `json:"gasPrice"`
	GasUsed     string `json:"gasUsed"`
	BlockNumber string `json:"blockNumber"`
}

type Payload struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

func getTransactionsByBlock(blockNumber *big.Int) ([]Transaction, error) {
	payload := Payload{
		Jsonrpc: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{"0x" + blockNumber.Text(16), true},
		ID:      1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(alchemyURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Result.Transactions, nil
}

func getGasUsedForTransaction(txHash string) (uint64, error) {
	// Prepare the payload for eth_getTransactionReceipt
	payload := Payload{
		Jsonrpc: "2.0",
		Method:  "eth_getTransactionReceipt",
		Params:  []interface{}{txHash},
		ID:      1,
	}
	payloadBytes, _ := json.Marshal(payload)

	resp, err := http.Post(alchemyURL, "application/json", strings.NewReader(string(payloadBytes)))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var response Response2
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return 0, err
	}

	// Extract the gasUsed field from the transaction receipt
	gasUsed, err := strconv.ParseUint(response.Result.GasUsed[2:], 16, 64)
	if err != nil {
		return 0, err
	}

	return gasUsed, nil
}

func getCheckpointEpoch(startBlock, endBlock uint64) []uint64 {
	allNumbers := make([]uint64, 0)
	for block := startBlock; block < endBlock; block++ {
		if block%EpochV2 == EpochV2-1 {
			allNumbers = append(allNumbers, block)
		}
	}
	return allNumbers
}

func doSomeStatistics(data []utils.ContractGasInfo) {
	fmt.Println("min", utils.CalculateMin(data))
	fmt.Println("max", utils.CalculateMax(data))
	fmt.Println("median", utils.CalculateMedian(data))
	fmt.Println("avg", utils.CalculateAverage(data))
}

func getRandomBlockNumbers(startBlock, endBlock, count uint64) []uint64 {
	allNumbers := make([]uint64, endBlock-startBlock+1)
	for i := range allNumbers {
		allNumbers[i] = startBlock + uint64(i)
	}

	rand.Shuffle(len(allNumbers), func(i, j int) {
		allNumbers[i], allNumbers[j] = allNumbers[j], allNumbers[i]
	})

	randomUniqueNumbers := allNumbers[:count]

	sort.Slice(randomUniqueNumbers, func(i, j int) bool {
		return randomUniqueNumbers[i] < randomUniqueNumbers[j]
	})

	result := make([]uint64, 0)
	for i, rn := range randomUniqueNumbers {
		if i % EpochV2 != EpochV2 - 1 {
			result = append(result, rn)
		}
	}

	return result
}

func main() {
	startBlock := uint64(29999998) // Replace with your start block number
	endBlock := uint64(31000000)   // Replace with your end block number
	// count := uint64(3000)


	// // For every block with random
	// blockList := getRandomBlockNumbers(startBlock, endBlock, count)

	blockList := getCheckpointEpoch(startBlock, endBlock)

	file, err := os.Create("contract_gas_usage.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, block := range blockList {
		transactions, err := getTransactionsByBlock(new(big.Int).SetUint64(block))
		if err != nil {
			fmt.Println("Error getting transactions for block:", block, err)
			continue
		}

		var totalGasUsedForContract map[string]uint64 = map[string]uint64 {
			roninValidatorSet: uint64(0),
			slashIndicator: uint64(0),
			stakingContract: uint64(0),
			profileContract: uint64(0),
			finalityTracking: uint64(0),
		}
		for _, tx := range transactions {
			switch tx.To {
			case roninValidatorSet, slashIndicator, profileContract:
				gasUsed, err := getGasUsedForTransaction(tx.Hash)
				if err != nil {
					fmt.Println("err roiii!!!!", "err", err)
					continue
				}
				totalGasUsedForContract[tx.To] += gasUsed
			default:
				continue
			}
		}
		for k, v := range(totalGasUsedForContract) {
			if v != 0 {
				line := fmt.Sprintf("Block: %d, Contract: %s, GasUsed: %d\n", block, k, v)
				_, err = writer.WriteString(line)
				if err != nil {
					panic(err)
				}
			}
		}
		fmt.Println(block)
	}

	writer.Flush()
	fmt.Println("Data written to contract_gas_usage.txt")






	// // =======================================================================//

	// // Open the file for reading
	// file, err := os.Open("contract_gas_usage.txt")
	// if err != nil {
	// 	fmt.Println("Error opening the file:", err)
	// 	return
	// }
	// defer file.Close()

	// // Create a scanner to read lines
	// scanner := bufio.NewScanner(file)


	// var (
	// 	prev = uint64(100)
	// 	totalGasUsed = uint64(0)
	// )

	// contractGasInfo := make([]utils.ContractGasInfo, 0)
	// totalGasUsedByBlock := make(map[uint64]uint64)

	// cc := make([]utils.ContractGasInfo, 0)
	// // Process each line
	// for scanner.Scan() {
	// 	line := scanner.Text()
	// 	// Split the line into its components
	// 	parts := strings.Split(line, ", ")
	// 	if len(parts) != 3 {
	// 		fmt.Println("Invalid line format:", line)
	// 		continue
	// 	}

	// 	block, err := strconv.ParseUint(strings.Split(parts[0], ": ")[1], 10, 64)
	// 	if err != nil {
	// 		fmt.Println("Error parsing block:", err)
	// 		continue
	// 	}

	// 	contract := strings.Split(parts[1], ": ")[1]

	// 	gasUsed, err := strconv.ParseUint(strings.Split(parts[2], ": ")[1], 10, 64)
	// 	if err != nil {
	// 		fmt.Println("Error parsing gasUsed:", err)
	// 		continue
	// 	}

	// 	if prev == 100 {
	// 		prev = block
	// 		totalGasUsed = gasUsed
	// 		continue
	// 	} else {
	// 		if prev == block {
	// 			totalGasUsed += gasUsed
	// 		} else {
	// 			prev = block
	// 			contractGasInfo = append(contractGasInfo, utils.ContractGasInfo{block, contract, gasUsed})
	// 			totalGasUsed = gasUsed
	// 		}
	// 	}

	// 	switch contract {
	// 	case roninValidatorSet, slashIndicator, profileContract:
	// 		totalGasUsedByBlock[block] += gasUsed
	// 	default:
	// 	}

	// 	// Now you have the components (block, contract, gasUsed) and can process them as needed
	// }



	// for k, v := range totalGasUsedByBlock {
	// 	cc = append(cc, utils.ContractGasInfo{k, "0x", v})
	// }



	// doSomeStatistics(cc)
}
