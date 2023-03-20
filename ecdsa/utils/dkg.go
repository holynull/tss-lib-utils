package utils

import (
	"errors"
	"runtime"
	"time"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/test"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/holynull/tss-lib-utils/tools"
	"golang.org/x/net/context"
)

func StartDKGParty(ctx context.Context, gid string, localParam *keygen.LocalPreParams, storedPartyIds tss.SortedPartyIDs, partyIndex int, partyCount int, threshold int, sendMsg func(tss.ParsedMessage, string) error) (tss.Party, chan keygen.LocalPartySaveData) {
	outCh := make(chan tss.Message)
	endCh := make(chan keygen.LocalPartySaveData)
	finalCh := make(chan keygen.LocalPartySaveData)
	mpcCtx := tss.NewPeerContext(storedPartyIds)
	params := tss.NewParameters(tss.S256(), mpcCtx, storedPartyIds[partyIndex], partyCount, threshold)
	partyObj := keygen.NewLocalParty(params, outCh, endCh, *localParam)
	go func() {
	Loop:
		for {
			select {
			case <-ctx.Done():
				Logger.Error(errors.New("DKG_TIME_OUT"))
				break Loop
			case message := <-outCh:
				Logger.Debug("DKG output data.")
				msg := message.(tss.ParsedMessage)
				err := sendMsg(msg, gid)
				if err != nil {
					Logger.Errorf("[3]: %v", err)
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

func HandleDKGMPCMessageFromOtherParty(thisParty tss.Party, tssMessage tss.Message, resultChan chan keygen.LocalPartySaveData) (*keygen.LocalPartySaveData, error) {
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
