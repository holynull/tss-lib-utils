package utils

import (
	"runtime"
	"time"

	"github.com/bnb-chain/tss-lib/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/eddsa/resharing"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/holynull/tss-lib-utils/tools"
	"golang.org/x/net/context"
)

func StartNewOrOldParty(ctx context.Context, mpcCtxIndexArr []int32, sk keygen.LocalPartySaveData, gid string, partyIndex int, isOld bool, partyCount int, threshold int, nPartyCount int, nThreshold int, sendMsg func(tss.ParsedMessage, string) error) (tss.Party, chan keygen.LocalPartySaveData) {
	storedOldPartyIds := tools.GenerateTestPartyIDsUsingInputRandomKey(sk.Ks[0], partyCount, 0)
	var oldParties []*tss.PartyID
	for i := 0; i < len(mpcCtxIndexArr); i++ {
		pid := storedOldPartyIds[mpcCtxIndexArr[i]]
		pid.Index = i
		oldParties = append(oldParties, pid)
	}
	var oldCtx *tss.PeerContext
	newPartyIDs := tools.GeneratePartyIDsUsingInputRandomKeyWithDefaultI(sk.Ks[0], nPartyCount, int(mpcCtxIndexArr[len(mpcCtxIndexArr)-1])+1)
	newCtx := tss.NewPeerContext(newPartyIDs)
	oldCtx = tss.NewPeerContext(oldParties)
	var partyId tss.PartyID
	if isOld {
		partyId = *oldParties[partyIndex]
	} else {
		partyId = *newPartyIDs[partyIndex]
	}
	params := tss.NewReSharingParameters(
		tss.S256(),
		oldCtx,
		newCtx,
		&partyId,
		partyCount,
		threshold,
		nPartyCount,
		nThreshold)
	save := keygen.NewLocalPartySaveData(partyCount)
	outCh := make(chan tss.Message)
	nEndCh := make(chan keygen.LocalPartySaveData)
	finalCh := make(chan keygen.LocalPartySaveData)
	var partyObj tss.Party
	if isOld {
		partyObj = resharing.NewLocalParty(params, sk, outCh, nEndCh)
	} else {
		partyObj = resharing.NewLocalParty(params, save, outCh, nEndCh)
	}
	go func() {
	Loop:
		for {
			select {
			case message := <-outCh:
				Logger.Debug("Resharing output data.")
				msg := message.(tss.ParsedMessage)
				err := sendMsg(msg, gid)
				if err != nil {
					Logger.Error(err)
				}
			case result := <-nEndCh:
				finalCh <- result
				break Loop
			default:
			}
			if runtime.GOOS == "js" {
				time.Sleep(time.Duration(tools.Elapsed) * time.Millisecond)
			}
		}
	}()
	return partyObj, finalCh
}
