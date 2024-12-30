# bitcoin-bruteforce

A Go program designed to create private keys, derive corresponding public keys from the private keys, and then check
that the generated wallet addresses have funds. This is the most recent up to date FREE bruteforcer.

# To Do

use database to store wallets
fix test with correct key address generated with bitcoin-cli

# how to use

go build bitcoin-wallet-bruteforce-offline.go

./bitcoin-wallet-bruteforce-offline.go threads out-file.txt btc-data-file.txt

Example: ./bitcoin-wallet-bruteforce-offline.go 1000000 out.txt btc_aa.txt

# Information

All bitcoin addresses with funds in them will be recorded to the out-file.txt you choose. You can also rename this to
anything you want. I advise you to run this in a screen and leave it for running for days on end. This is an efficient
method of trying to obtain free funds.

Make sure Golang 1.2.1 is installed or latest version.

The scripts come with the option to use telegram bots to save any bitcoin wallets automatically. If you do not whish to
use this feature then put 123 as both values for the chat id and bot token.

# Test (not working) (todo)

bitcoin-cli getnewaddress # P2PKH address
bitcoin-cli getnewaddress "" "p2sh-segwit" # P2SH-SegWit
bitcoin-cli getnewaddress "" "bech32" # Native SegWit
bitcoin-cli dumpprivkey <address> # Get private key
