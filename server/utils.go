package server

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/unicornultrafoundation/go-u2u/accounts/abi"
	"github.com/unicornultrafoundation/go-u2u/common"
)

type Receipt struct {
	Logs []Log `json:"logs"`
}

// Log represents a single log entry in the receipt
type Log struct {
	Topics []string `json:"topics"`
	Data   string   `json:"data"`
}

func DecodeContractEvent(contractABI string, receiptJSON string, eventName string, output interface{}) error {
	fmt.Printf("Starting DecodeContractEvent with eventName: %s\n", eventName)
	fmt.Printf("Output type: %T\n", output)

	// Parse the receipt JSON
	var receipt Receipt
	if err := json.Unmarshal([]byte(receiptJSON), &receipt); err != nil {
		return fmt.Errorf("failed to parse receipt JSON: %v", err)
	}
	fmt.Printf("Found %d logs in receipt\n", len(receipt.Logs))

	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return fmt.Errorf("failed to parse ABI: %v", err)
	}
	fmt.Printf("ABI parsed successfully, looking for event: %s\n", eventName)

	// Find the event in the receipt logs
	for i, log := range receipt.Logs {
		fmt.Printf("\nProcessing log %d:\n", i)
		fmt.Printf("Topics: %v\n", log.Topics)
		fmt.Printf("Data: %s\n", log.Data)

		// Get the event ID (topic0) for the specified event name
		event, exists := parsedABI.Events[eventName]
		if !exists {
			return fmt.Errorf("event %s not found in ABI", eventName)
		}
		fmt.Printf("Event found in ABI with ID: %s\n", event.ID.Hex())

		// Convert topics from hex strings to common.Hash
		topics := make([]common.Hash, len(log.Topics))
		for i, topic := range log.Topics {
			topics[i] = common.HexToHash(topic)
		}
		fmt.Printf("Converted topics: %v\n", topics)

		// Check if this log corresponds to our event
		if len(topics) > 0 && topics[0] == event.ID {
			fmt.Printf("Found matching event log!\n")

			// Convert data from hex string to bytes
			data := common.FromHex(log.Data)
			fmt.Printf("Event data bytes: %x\n", data)

			// Create a map to hold all event data
			eventData := make(map[string]interface{})

			// Unpack the non-indexed data into the map
			err = parsedABI.UnpackIntoMap(eventData, eventName, data)
			if err != nil {
				return fmt.Errorf("failed to decode event data: %v", err)
			}
			fmt.Printf("Successfully unpacked non-indexed data: %v\n", eventData)

			// If the event has indexed parameters, add them to the map
			if len(topics) > 1 {
				fmt.Printf("Processing indexed parameters...\n")
				indexedArgs := make(abi.Arguments, 0)
				for _, arg := range event.Inputs {
					if arg.Indexed {
						indexedArgs = append(indexedArgs, arg)
					}
				}
				fmt.Printf("Found %d indexed arguments\n", len(indexedArgs))

				if len(indexedArgs) > 0 {
					err = abi.ParseTopicsIntoMap(eventData, indexedArgs, topics[1:])
					if err != nil {
						return fmt.Errorf("failed to decode indexed event parameters: %v", err)
					}
					fmt.Printf("Combined event data: %v\n", eventData)
				}
			}

			// Convert the map to JSON and unmarshal into the output struct
			jsonData, err := json.Marshal(eventData)
			if err != nil {
				return fmt.Errorf("failed to marshal event data: %v", err)
			}

			if err := json.Unmarshal(jsonData, output); err != nil {
				return fmt.Errorf("failed to unmarshal into output struct: %v", err)
			}

			return nil // Successfully decoded the event
		}
	}

	return fmt.Errorf("event %s not found in transaction logs", eventName)
}

// Example usage struct
type SpinCompletedEvent struct {
	Segment   uint8
	Prize     *big.Int
	Player    common.Address
	Timestamp *big.Int
}
