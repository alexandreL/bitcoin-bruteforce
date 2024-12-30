package main

import (
	"encoding/hex"
	"os"
	"testing"
)

func TestCreateAddresses(t *testing.T) {
	pubKey, _ := hex.DecodeString("0250863ad64a87ae8a2fe83c1af1a8403cb53f53e486d8511dad8a04887e5b2352")

	tests := []struct {
		name     string
		pubKey   []byte
		addrType AddressType
		want     string
		wantErr  bool
	}{
		{"Valid P2PKH", pubKey, P2PKH, "1PMycacnJaSqwwJqjawXBErnLsZ7RkXUAs", false},
		{"Valid P2SH", pubKey, P2SH_SEGWIT, "3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy", false},
		{"Valid Native SegWit", pubKey, NATIVE_SEGWIT, "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq", false},
		{"Invalid pubkey", []byte{}, P2PKH, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createBitcoinAddress(tt.pubKey, tt.addrType)
			if (err != nil) != tt.wantErr {
				t.Errorf("createBitcoinAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("createBitcoinAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadAddresses(t *testing.T) {
	// Create temp test files
	validFile := "valid.txt"
	emptyFile := "empty.txt"

	os.WriteFile(validFile, []byte("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa\n"), 0644)
	os.WriteFile(emptyFile, []byte(""), 0644)

	tests := []struct {
		name    string
		file    string
		want    int
		wantErr bool
	}{
		{"Valid file", validFile, 1, false},
		{"Empty file", emptyFile, 0, false},
		{"Non-existent file", "bad.txt", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readAddresses(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("readAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("readAddresses() = %v addresses, want %v", len(got), tt.want)
			}
		})
	}

	// Cleanup
	os.Remove(validFile)
	os.Remove(emptyFile)
}

func TestGenerateKeyAndAddress(t *testing.T) {
	privKey, addrs, err := generateKeyAndAddress()
	if err != nil {
		t.Fatalf("generateKeyAndAddress() error = %v", err)
	}

	if len(privKey) != 64 {
		t.Errorf("Private key length = %v, want 64", len(privKey))
	}

	if len(addrs.p2pkh) == 0 || len(addrs.p2sh) == 0 || len(addrs.bech32) == 0 {
		t.Error("One or more addresses are empty")
	}
}

func TestBtcAddressExist(t *testing.T) {
	addrs := BtcAddress{
		p2pkh:  "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
		p2sh:   "3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy",
		bech32: "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
	}

	tests := []struct {
		name       string
		addresses  map[string]bool
		want       string
		wantExists bool
	}{
		{
			"Address exists",
			map[string]bool{"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa": true},
			"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			true,
		},
		{
			"Address doesn't exist",
			map[string]bool{},
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, exists := btcAddressExist(addrs, tt.addresses)
			if exists != tt.wantExists {
				t.Errorf("btcAddressExist() exists = %v, want %v", exists, tt.wantExists)
			}
			if got != tt.want {
				t.Errorf("btcAddressExist() = %v, want %v", got, tt.want)
			}
		})
	}
}
