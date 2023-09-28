package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/izuc/zipp/client/wallet"
	"github.com/izuc/zipp/client/wallet/packages/destroynftoptions"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/vm/devnetvm"
)

func execDestroyNFTCommand(command *flag.FlagSet, cliWallet *wallet.Wallet) {
	command.Usage = func() {
		printUsage(command)
	}

	helpPtr := command.Bool("help", false, "show this help screen")
	nftIDPtr := command.String("id", "", "unique identifier of the nft that should be destroyed")
	accessManaPledgeIDPtr := command.String("access-mana-id", "", "node ID to pledge access mana to")
	consensusManaPledgeIDPtr := command.String("consensus-mana-id", "", "node ID to pledge consensus mana to")

	err := command.Parse(os.Args[2:])
	if err != nil {
		panic(err)
	}

	if *helpPtr {
		printUsage(command)
	}

	if *nftIDPtr == "" {
		printUsage(command, "an nft (alias) ID must be given for destroy")
	}

	aliasID, err := devnetvm.AliasAddressFromBase58EncodedString(*nftIDPtr)
	if err != nil {
		printUsage(command, err.Error())
	}
	fmt.Println("Destroying NFT...")
	_, err = cliWallet.DestroyNFT(
		destroynftoptions.Alias(aliasID.Base58()),
		destroynftoptions.AccessManaPledgeID(*accessManaPledgeIDPtr),
		destroynftoptions.ConsensusManaPledgeID(*consensusManaPledgeIDPtr),
	)
	if err != nil {
		printUsage(command, err.Error())
	}

	fmt.Println()
	fmt.Println("Destroying NFT... [DONE]")
}
