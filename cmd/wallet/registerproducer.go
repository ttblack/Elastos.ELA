package wallet

import (
	"fmt"
	"os"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"

	"github.com/urfave/cli"
)

var registerproducer = cli.Command {
	Name:  "registerproducer",
	Usage: "Build a tx to register producer",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionNodePublicKeyFlag,
		cmdcom.TransactionNickNameFlag,
		cmdcom.TransactionUrlFlag,
		cmdcom.TransactionLocationFlag,
		cmdcom.TransactionNetAddressFlag,
		cmdcom.TransactionStakeUntilFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createRegisterProducerTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func createRegisterProducerTransaction(c *cli.Context) error {
	return createProducerInfoCommonTransaction(c, common2.RegisterProducer, true)
}
