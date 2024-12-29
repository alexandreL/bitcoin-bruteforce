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

// Version byte for mainnet = 0x00
const VERSION_BYTE = 0x00

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

func publicKeyToAddress(pubKey []byte) (string, error) {

	// 1. SHA256
	sha256Hash := sha256.Sum256(pubKey)

	// 2. RIPEMD160
	ripemd160Hash := ripemd160.New()
	_, err := ripemd160Hash.Write(sha256Hash[:])
	if err != nil {
		return "", err
	}
	ripemdHash := ripemd160Hash.Sum(nil)

	// 3. Add version byte
	versionedHash := append([]byte{VERSION_BYTE}, ripemdHash...)

	// 4. Double SHA256 for checksum
	firstSHA := sha256.Sum256(versionedHash)
	secondSHA := sha256.Sum256(firstSHA[:])

	// 5. First 4 bytes of double-SHA is checksum
	checksum := secondSHA[:4]

	// 6. Concat version + ripemdHash + checksum
	finalHash := append(versionedHash, checksum...)

	// 7. Base58 encode
	address := base58.Encode(finalHash)

	return address, nil
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
