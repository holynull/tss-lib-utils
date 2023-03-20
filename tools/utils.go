package tools

import (
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/tss"
)

const Elapsed = 20 // ms

func GenerateTestPartyIDsUsingInputRandomKey(key *big.Int, count int, startAt ...int) tss.SortedPartyIDs {
	ids := make(tss.UnSortedPartyIDs, 0, count)
	frm := 0
	i := 0 // default `i`
	if len(startAt) > 0 {
		frm = startAt[0]
		i = startAt[0]
	}
	for ; i < count+frm; i++ {

		extraNum := big.NewInt(int64(i))
		ids = append(ids, &tss.PartyID{
			MessageWrapper_PartyID: &tss.MessageWrapper_PartyID{
				Id:      fmt.Sprintf("%d", i+1),
				Moniker: fmt.Sprintf("P[%d]", i+1),
				Key:     new(big.Int).Add(key, extraNum).Bytes(),
			},
			Index: i,
			// this key makes tests more deterministic
		})
	}
	return tss.SortPartyIDs(ids, startAt...)
}

func GeneratePartyIDsUsingInputRandomKeyWithDefaultI(key *big.Int, count int, defaultI int) tss.SortedPartyIDs {
	ids := make(tss.UnSortedPartyIDs, 0, count)
	frm := defaultI
	i := defaultI

	for ; i < count+frm; i++ {

		extraNum := big.NewInt(int64(i))
		ids = append(ids, &tss.PartyID{
			MessageWrapper_PartyID: &tss.MessageWrapper_PartyID{
				Id:      fmt.Sprintf("%d", i+1),
				Moniker: fmt.Sprintf("P[%d]", i+1),
				Key:     new(big.Int).Add(key, extraNum).Bytes(),
			},
			Index: i,
			// this key makes tests more deterministic
		})
	}
	return tss.SortPartyIDs(ids)
}
