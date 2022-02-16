// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package outputpayload

import (
	"errors"
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

const ExchangeVoteOutputVersion byte = 0x00

// CandidateVotes defines the voting information for individual candidates.
type ExchangeVoteOutput struct {
	Version      byte
	StakeAddress common.Uint168
	Votes        common.Fixed64
}

func (ev *ExchangeVoteOutput) Data() []byte {
	return nil
}

func (ev *ExchangeVoteOutput) Serialize(w io.Writer) error {
	if _, err := w.Write([]byte{ev.Version}); err != nil {
		return err
	}
	if err := ev.StakeAddress.Serialize(w); err != nil {
		return err
	}
	if err := ev.Votes.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (ev *ExchangeVoteOutput) Deserialize(r io.Reader) error {
	version, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	ev.Version = version[0]
	if err := ev.StakeAddress.Deserialize(r); err != nil {
		return err
	}
	if err := ev.Votes.Deserialize(r); err != nil {
		return err
	}

	return nil
}


func (ev *ExchangeVoteOutput) GetVersion() byte {
	return ev.Version
}


func (ev *ExchangeVoteOutput) Validate() error {
	if ev == nil {
		return errors.New("exchange vote output payload is nil")
	}
	if ev.Version > ExchangeVoteOutputVersion {
		return errors.New("invalid exchange vote version")
	}

	return nil
}

func (ev *ExchangeVoteOutput) String() string {
	addr, _ := ev.StakeAddress.ToAddress()
	return fmt.Sprint("{\n\t\t\t\t",
		"StakeAddress: ", addr, "\n\t\t\t\t",
		"Votes: ", ev.Votes, "}\n\t\t\t\t")
}
