//offline version, use any database u want to achieve this, I used http://alladdresses.loyce.club/

package main

import (
	"btcgen/telegram"
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"log"
	"os"
	"strconv"
	"sync"

	"crypto/sha256"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/ripemd160"
)

// Version byte for mainnet = 0x00
const VERSION_BYTE = 0x00

const (
	P2PKH_VERSION = 0x00 // Mainnet P2PKH
	P2SH_VERSION  = 0x05 // Mainnet P2SH
	BECH32_HRP    = "bc" // Mainnet Bech32
)

type AddressType int

const (
	P2PKH AddressType = iota
	P2SH_SEGWIT
	NATIVE_SEGWIT
)

type BtcAddress struct {
	p2pkh  string
	p2sh   string
	bech32 string
}
type Counter struct {
	count int64
	mutex sync.Mutex
}

func (c *Counter) Increment() {
	c.mutex.Lock()
	c.count++
	c.mutex.Unlock()
}

func (c *Counter) GetCount() int64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.count
}
func createBitcoinAddress(pubKey []byte, addrType AddressType) (string, error) {
	switch addrType {
	case P2PKH:
		return createP2PKHAddress(pubKey)
	case P2SH_SEGWIT:
		return createP2SHSegWitAddress(pubKey)
	case NATIVE_SEGWIT:
		return createNativeSegWitAddress(pubKey)
	default:
		return "", errors.New("invalid address type")
	}
}

func createP2PKHAddress(pubKey []byte) (string, error) {
	// SHA256 then RIPEMD160
	hash := hash160(pubKey)

	// Add version byte
	versioned := append([]byte{P2PKH_VERSION}, hash...)

	// Add checksum
	checksum := doubleSHA256(versioned)[:4]
	full := append(versioned, checksum...)

	return base58.Encode(full), nil
}

func createP2SHSegWitAddress(pubKey []byte) (string, error) {
	// SegWit program: version 0 + 20-byte pubkey hash
	hash := hash160(pubKey)
	program := append([]byte{0x00}, hash...)

	// Script: OP_0 + push(program)
	redeemScript := append([]byte{0x00, 0x14}, program...)

	// P2SH: hash160(redeemScript)
	scriptHash := hash160(redeemScript)

	// Add P2SH version
	versioned := append([]byte{P2SH_VERSION}, scriptHash...)
	checksum := doubleSHA256(versioned)[:4]
	full := append(versioned, checksum...)

	return base58.Encode(full), nil
}

func createNativeSegWitAddress(pubKey []byte) (string, error) {
	program := hash160(pubKey)

	// Bech32 encoding with witness version 0
	conv, err := bech32.ConvertBits(program, 8, 5, true)
	if err != nil {
		return "", err
	}

	combined := append([]byte{0x00}, conv...)
	address, err := bech32.Encode(BECH32_HRP, combined)
	if err != nil {
		return "", err
	}

	return address, nil
}

// Helper functions
func hash160(data []byte) []byte {
	sha := sha256.Sum256(data)
	ripe := ripemd160.New()
	ripe.Write(sha[:])
	return ripe.Sum(nil)
}

func doubleSHA256(data []byte) []byte {
	hash1 := sha256.Sum256(data)
	hash2 := sha256.Sum256(hash1[:])
	return hash2[:]
}

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

func generateKeyAndAddress() (string, BtcAddress, error) {
	privateKey, err := btcec.NewPrivateKey()

	if err != nil {
		return "", BtcAddress{}, err
	}

	serializedPublicKey := privateKey.PubKey().SerializeCompressed()
	bitcoinAddressP2PKH, err := createBitcoinAddress(serializedPublicKey, P2PKH)
	if err != nil {
		return "", BtcAddress{}, err
	}
	bitcoinAddressP2SH, err := createBitcoinAddress(serializedPublicKey, P2SH_SEGWIT)
	if err != nil {
		return "", BtcAddress{}, err
	}
	bitcoinAddressBech32, err := createBitcoinAddress(serializedPublicKey, NATIVE_SEGWIT)
	if err != nil {
		return "", BtcAddress{}, err
	}

	return hex.EncodeToString(privateKey.Serialize()), BtcAddress{bitcoinAddressP2PKH, bitcoinAddressP2SH, bitcoinAddressBech32}, nil
}

func btcAddressExist(address BtcAddress, btcAddresses map[string]bool) (string, bool) {
	if btcAddresses[address.p2pkh] {
		return address.p2pkh, true
	}
	if btcAddresses[address.p2sh] {
		return address.p2sh, true
	}
	if btcAddresses[address.bech32] {
		return address.bech32, true
	}

	return "", false
}

func worker(id int, wg *sync.WaitGroup, mutex *sync.Mutex, outputFile string, btcAddresses map[string]bool, counter *Counter, telegramBot *telegram.Bot) {
	defer wg.Done()

	for {
		counter.Increment()
		if counter.GetCount()%1000 == 0 {
			fmt.Printf("Checked %d addresses\n", counter.GetCount())
		}
		privateKey, publicAddress, err := generateKeyAndAddress()
		if err != nil {
			log.Printf("Worker %d: Failed to generate key and address: %s", id, err)
			continue
		}

		if publicAddressKey, exists := btcAddressExist(publicAddress, btcAddresses); exists {
			fmt.Printf("Match Found! Privatekey: %s Publicaddress: %s\n", privateKey, publicAddressKey)

			mutex.Lock()
			file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("Worker %d: Failed to open file: %s", id, err)
				mutex.Unlock()
				continue
			}

			if _, err := file.WriteString(fmt.Sprintf("%s:%s\n", privateKey, publicAddressKey)); err != nil {
				log.Printf("Worker %d: Failed to write to file: %s", id, err)
			}

			telegramBot.SendMessage(fmt.Sprintf("Match Found! Privatekey: %s Publicaddress: %s", privateKey, publicAddressKey))
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
	// Load environment variables
	log.Println("Loading environment variables...")
	err = godotenv.Load()
	if err != nil {
		log.Println("Error loading environment variables.")
	}
	log.Println("Environment variables loaded successfully.")

	// Initialize Telegram bot
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	telegramBot := telegram.NewBot(botToken, chatID)
	log.Println("Telegram bot initialized.")
	outputFile := os.Args[2]
	btcAddressesFile := os.Args[3]

	btcAddresses, err := readAddresses(btcAddressesFile)
	if err != nil {
		log.Fatalf("Failed to read BTC addresses: %s", err)
	}

	var wg sync.WaitGroup
	var mutex sync.Mutex
	counter := &Counter{}

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go worker(i, &wg, &mutex, outputFile, btcAddresses, counter, telegramBot)
	}

	wg.Wait()
}
