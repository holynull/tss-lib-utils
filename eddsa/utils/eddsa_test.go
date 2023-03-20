package utils

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/bnb-chain/tss-lib/eddsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
)

func RandStr(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	rand.Seed(time.Now().UnixNano() + int64(rand.Intn(100)))
	for i := 0; i < length; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func TestDKG(t *testing.T) {
	partyIDs := tss.GenerateTestPartyIDs(2, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	gid := RandStr(32)
	recvCh := make([]chan tss.Message, len(partyIDs))
	for i := range partyIDs {
		rch := make(chan tss.Message)
		recvCh[i] = rch
	}
	var result []*keygen.LocalPartySaveData
	wg := sync.WaitGroup{}
	wg.Add(len(partyIDs))
	for i := range partyIDs {
		go func(i int) {
			defer func() {
				wg.Done()
			}()
			sendMsg := func(msg tss.ParsedMessage, gid string) error {
				if msg.GetTo() == nil {
					for pid := range recvCh {
						t.Logf("Send message: to party %d, type %s", pid, msg.Type())
						recvCh[pid] <- msg
						t.Log("Send!")
					}
				} else {
					for _, to := range msg.GetTo() {
						t.Logf("Send message: to party %d, type %s", to.Index, msg.Type())
						recvCh[to.Index] <- msg
						t.Log("Send!")
					}
				}
				return nil
			}
			party, fch := StartDKGParty(ctx, gid, partyIDs, i, 2, 1, sendMsg)
			doneCh := make(chan string)
		Loop:
			for {
				select {
				case msg := <-recvCh[i]:
					t.Logf("Party %d Receive message: from party %d, type %s", i, msg.GetFrom().Index, msg.Type())
					go func() {
						sk, err := HandleDKGMPCMessageFromOtherParty(party, msg, fch)
						if err != nil {
							t.Error(err)
							doneCh <- "done"
							return
						}
						if sk != nil {
							result = append(result[:], sk)
							t.Logf("SK: %v", sk)
							doneCh <- "done"
						}
					}()
				case <-doneCh:
					break Loop
				default:
					continue Loop
				}
			}
		}(i)
	}
	wg.Wait()
}
