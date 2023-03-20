package utils

import (
	"context"
	"errors"
	"math/big"
	"runtime"
	"time"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/eddsa/signing"
	"github.com/bnb-chain/tss-lib/test"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/holynull/tss-lib-utils/tools"
)

func StartSignParty(ctx context.Context, msg big.Int, mpcCtxIndexArr []int32, sk *keygen.LocalPartySaveData, gid string, partyIndex int, partyCount int, threshold int, sendMsg func(tss.ParsedMessage, string) error) (tss.Party, chan common.SignatureData) {
	outCh := make(chan tss.Message)
	storedPartyIds := tools.GenerateTestPartyIDsUsingInputRandomKey(sk.Ks[0], partyCount, 0)
	var partyIds []tss.PartyID
	for _, index := range mpcCtxIndexArr {
		partyIds = append(partyIds[:], *storedPartyIds[index])
	}
	mpcCtx := tss.NewPeerContext(storedPartyIds[:2])
	params := tss.NewParameters(tss.Edwards(), mpcCtx, storedPartyIds[partyIndex], partyCount, threshold)
	endCh := make(chan common.SignatureData)
	finalCh := make(chan common.SignatureData)
	partyObj := signing.NewLocalParty(&msg, params, *sk, outCh, endCh)
	go func() {
	Loop:
		for {
			select {
			case <-ctx.Done():
				Logger.Error(errors.New("SIGN_TIME_OUT"))
				break Loop
			case message := <-outCh:
				Logger.Debug("Sign output data.")
				msg := message.(tss.ParsedMessage)
				err := sendMsg(msg, gid)
				if err != nil {
					Logger.Error(err)
				}
			case result := <-endCh:
				finalCh <- result
				break Loop
			default:
			}
			if runtime.GOOS == "js" {
				time.Sleep(time.Duration(tools.Elapsed) * time.Millisecond)
			}
		}
	}()
	partyObj.Start()
	return partyObj, finalCh
}

func HandleSignMPCMessageFromOtherParty(thisParty tss.Party, tssMessage tss.Message, resultChan chan common.SignatureData) (*common.SignatureData, error) {
	errCh := make(chan *tss.Error)
	test.SharedPartyUpdater(
		thisParty,
		tssMessage,
		errCh)
	select {
	case result := <-resultChan:
		return &result, nil
	case err := <-errCh:
		Logger.Error(err)
		return nil, err
	default:
		return nil, nil
	}
}
