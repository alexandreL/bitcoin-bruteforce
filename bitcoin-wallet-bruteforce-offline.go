//offline version, use any database u want to achieve this, I used http://alladdresses.loyce.club/

package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/base58"
	"log"
	"os"
	"strconv"
	"sync"

	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
)

func readAddresses(filePath string) (map[string]bool, error) {
	addresses := make(map[string]bool)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		addresses[scanner.Text()] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return addresses, nil
}

func generateKeyAndAddress() (string, string, error) {
	privateKey, err := btcec.NewPrivateKey()

	if err != nil {
		return "", "", err
	}

	serializedPublicKey := privateKey.PubKey().SerializeCompressed()
	address, err := publicKeyToAddress(serializedPublicKey)

	if err != nil {
		return "", "", err
	}
	return hex.EncodeToString(privateKey.Serialize()), address, nil
}

func publicKeyToAddress(serializedPublicKey []byte) (string, error) {
	publicSHA256 := sha256.Sum256(serializedPublicKey)

	ripemd160Hasher := ripemd160.New()
	_, err := ripemd160Hasher.Write(publicSHA256[:])
	if err != nil {
		return "", err
	}

	publicRIPEMD160 := ripemd160Hasher.Sum(nil)

	return base58.CheckEncode(publicRIPEMD160[:], 0x00), nil
}

func worker(id int, wg *sync.WaitGroup, mutex *sync.Mutex, outputFile string, btcAddresses map[string]bool) {
	defer wg.Done()

	for {
		privateKey, publicAddress, err := generateKeyAndAddress()
		if err != nil {
			log.Printf("Worker %d: Failed to generate key and address: %s", id, err)
			continue
		}

		if _, exists := btcAddresses[publicAddress]; exists {
			fmt.Printf("Match Found! Privatekey: %s Publicaddress: %s\n", privateKey, publicAddress)

			mutex.Lock()
			file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("Worker %d: Failed to open file: %s", id, err)
				mutex.Unlock()
				continue
			}

			if _, err := file.WriteString(fmt.Sprintf("%s:%s\n", privateKey, publicAddress)); err != nil {
				log.Printf("Worker %d: Failed to write to file: %s", id, err)
			}
			file.Close()
			mutex.Unlock()
		}

	}
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: ./golangscript <threads> <output-file.txt> <btc-address-file.txt>")
		os.Exit(1)
	}

	numThreads, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid number of threads: %s", err)
	}

	outputFile := os.Args[2]
	btcAddressesFile := os.Args[3]

	btcAddresses, err := readAddresses(btcAddressesFile)
	if err != nil {
		log.Fatalf("Failed to read BTC addresses: %s", err)
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go worker(i, &wg, &mutex, outputFile, btcAddresses)
	}

	wg.Wait()
}
