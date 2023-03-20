package utils

import (
	"errors"
	"runtime"
	"time"

	"github.com/bnb-chain/tss-lib/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/test"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/holynull/tss-lib-utils/tools"
	"golang.org/x/net/context"
)

func StartDKGParty(ctx context.Context, gid string, storedPartyIds tss.SortedPartyIDs, partyIndex int, partyCount int, threshold int, sendMsg func(tss.ParsedMessage, string) error) (tss.Party, chan keygen.LocalPartySaveData) {
	outCh := make(chan tss.Message)
	endCh := make(chan keygen.LocalPartySaveData)
	finalCh := make(chan keygen.LocalPartySaveData)
	mpcCtx := tss.NewPeerContext(storedPartyIds)
	params := tss.NewParameters(tss.Edwards(), mpcCtx, storedPartyIds[partyIndex], partyCount, threshold)
	partyObj := keygen.NewLocalParty(params, outCh, endCh)
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
				// if message.GetTo() == nil {
				// 	for _, P := range storedPartyIds {
				// 		if P.Index == msg.GetFrom().Index {
				// 			continue
				// 		}
				err := sendMsg(msg, gid)
				if err != nil {
					Logger.Error(err)
				}
				// 	}
				// } else {
				// 	if message.GetTo()[0].Index == msg.GetFrom().Index {
				// 		Logger.Error("party %d tried to send a message to itself (%d)", message.GetTo()[0].Index, msg.GetFrom().Index)
				// 		continue
				// 	}
				// 	err := sendMsg(msg, gid)
				// 	if err != nil {
				// 		Logger.Error(err)
				// 	}
				// }

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
