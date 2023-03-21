package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
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
			preParam, err := keygen.GeneratePreParams(2 * time.Minute)
			if err != nil {
				t.Error(err)
				return
			}
			party, fch := StartDKGParty(ctx, gid, preParam, partyIDs, i, 2, 1, sendMsg)
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
							skJsonBytes, err := json.Marshal(sk)
							if err != nil {
								t.Error(err)
								return
							}
							path, err := os.Getwd()
							if err != nil {
								t.Error(err)
								return
							}
							fileName := fmt.Sprintf("sk_%d.json", i)
							fullName := fmt.Sprintf("%s/%s", path, fileName)
							err = os.WriteFile(fullName, skJsonBytes, os.ModePerm)
							if err != nil {
								t.Error(err)
								return
							}
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

func TestSign(t *testing.T) {
	partyCount := 2
	path, err := os.Getwd()
	if err != nil {
		t.Error(err)
		return
	}
	unSignedMsg := new(big.Int).SetBytes([]byte(RandStr(32)))
	wg := sync.WaitGroup{}
	wg.Add(partyCount)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	gid := RandStr(32)
	recvCh := make([]chan tss.Message, partyCount)
	for i := 0; i < partyCount; i++ {
		rch := make(chan tss.Message)
		recvCh[i] = rch
	}
	for i := 0; i < partyCount; i++ {
		go func(pIndex int) { // run a party
			defer func() {
				wg.Done()
			}()
			fileName := fmt.Sprintf("sk_%d.json", pIndex)
			fullName := fmt.Sprintf("%s/%s", path, fileName)
			skJsonBytes, err := os.ReadFile(fullName)
			if err != nil {
				t.Error(err)
				return
			}
			var sk keygen.LocalPartySaveData
			err = json.Unmarshal(skJsonBytes, &sk)
			if err != nil {
				t.Error(err)
				return
			}
			sendMsg := func(msg tss.ParsedMessage, gid string) error {
				time.Sleep(20 * time.Millisecond)
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
			var partyObj tss.Party
			var fch chan common.SignatureData
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				doneCh := make(chan string)
			Loop:
				for {
					select {
					case msg := <-recvCh[pIndex]:
						t.Logf("Party %d Receive message: from party %d, type %s", pIndex, msg.GetFrom().Index, msg.Type())
						go func() {
							signedData, err := HandleSignMPCMessageFromOtherParty(partyObj, msg, fch)
							if err != nil {
								t.Error(err)
								doneCh <- "done"
								return
							}
							if signedData != nil {
								signedDataJsonBytes, err := json.Marshal(signedData)
								if err != nil {
									t.Error(err)
									return
								}
								path, err := os.Getwd()
								if err != nil {
									t.Error(err)
									return
								}
								fileName := fmt.Sprintf("signed_data_%d.json", pIndex)
								fullName := fmt.Sprintf("%s/%s", path, fileName)
								err = os.WriteFile(fullName, signedDataJsonBytes, os.ModePerm)
								if err != nil {
									t.Error(err)
									return
								}
								t.Logf("Signed Data: %v", signedData)
								doneCh <- "done"
							}
						}()
					case <-doneCh:
						break Loop
					default:
						continue Loop
					}
				}
			}()
			partyObj, fch = StartSignParty(ctx, *unSignedMsg, []int32{0, 1}, &sk, gid, pIndex, partyCount, 1, sendMsg)
			wg.Wait()
		}(i)
	}
	wg.Wait()
}

func TestResharing(t *testing.T) {
	path, err := os.Getwd()
	if err != nil {
		t.Error(err)
		return
	}
	partyCount := 2
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(partyCount * 2)
	oRecvCh := make([]chan tss.Message, partyCount*2)
	for i := 0; i < partyCount; i++ {
		rch := make(chan tss.Message)
		oRecvCh[i] = rch
	}
	nRecvCh := make([]chan tss.Message, partyCount*2)
	for i := 0; i < partyCount; i++ {
		rch := make(chan tss.Message)
		nRecvCh[i] = rch
	}
	gid := RandStr(32)
	nDoneCh := make(chan string)
	oDoneCh := make(chan string)
	for i := 0; i < partyCount; i++ { // old party
		go func(pIndex int) {
			defer wg.Done()
			fileName := fmt.Sprintf("sk_%d.json", pIndex)
			fullName := fmt.Sprintf("%s/%s", path, fileName)
			skJsonBytes, err := os.ReadFile(fullName)
			if err != nil {
				t.Error(err)
				return
			}
			var sk keygen.LocalPartySaveData
			err = json.Unmarshal(skJsonBytes, &sk)
			if err != nil {
				t.Error(err)
				return
			}
			sendMsg := func(msg tss.ParsedMessage, gid string) error {
				if msg.IsToOldCommittee() { // to old
					for i, to := range msg.GetTo() {
						t.Logf("Send message: to party %d, type %s", i, msg.Type())
						oRecvCh[to.Index] <- msg
						t.Log("Send!")
					}
				}
				if !msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() { // to new
					for i, to := range msg.GetTo() {
						t.Logf("Send message: to party %d, type %s", i, msg.Type())
						nRecvCh[to.Index] <- msg
						t.Log("Send!")
					}
				}
				msg.GetTo()
				return nil
			}
			partyObj, fch := StartNewOrOldParty(ctx, []int32{0, 1}, nil, &sk, gid, pIndex, true, partyCount, 1, partyCount, 1, sendMsg)
		Loop:
			for {
				select {
				case msg := <-oRecvCh[pIndex]:
					t.Logf("Party %d Receive message: from party %d, type %s", pIndex, msg.GetFrom().Index, msg.Type())
					go func() {
						_, err := HandleDKGMPCMessageFromOtherParty(partyObj, msg, fch)
						if err != nil {
							t.Error(err)
							oDoneCh <- "done"
							nDoneCh <- "done"
							return
						}
					}()
				case <-oDoneCh:
					break Loop
				default:
					continue Loop
				}
			}
		}(i)
	}
	for i := 0; i < partyCount; i++ { // new party
		go func(pIndex int) {
			defer wg.Done()
			fileName := fmt.Sprintf("sk_%d.json", pIndex)
			fullName := fmt.Sprintf("%s/%s", path, fileName)
			skJsonBytes, err := os.ReadFile(fullName)
			if err != nil {
				t.Error(err)
				return
			}
			var sk keygen.LocalPartySaveData
			err = json.Unmarshal(skJsonBytes, &sk)
			if err != nil {
				t.Error(err)
				return
			}
			sendMsg := func(msg tss.ParsedMessage, gid string) error {
				if msg.IsToOldCommittee() { // to old
					for i, to := range msg.GetTo() {
						t.Logf("Send message: to party %d, type %s", i, msg.Type())
						oRecvCh[to.Index] <- msg
						t.Log("Send!")
					}
				}
				if !msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() { // to new
					for i, to := range msg.GetTo() {
						t.Logf("Send message: to party %d, type %s", i, msg.Type())
						nRecvCh[to.Index] <- msg
						t.Log("Send!")
					}
				}
				msg.GetTo()
				return nil
			}
			preParam, err := keygen.GeneratePreParams(2 * time.Minute)
			if err != nil {
				t.Error(err)
				return
			}
			partyObj, fch := StartNewOrOldParty(ctx, []int32{0, 1}, preParam, &sk, gid, pIndex, false, partyCount, 1, partyCount, 1, sendMsg)
		Loop:
			for {
				select {
				case msg := <-nRecvCh[pIndex]:
					t.Logf("Party %d Receive message: from party %d, type %s", pIndex, msg.GetFrom().Index, msg.Type())
					go func() {
						sk, err := HandleDKGMPCMessageFromOtherParty(partyObj, msg, fch)
						if err != nil {
							t.Error(err)
							oDoneCh <- "Done"
							nDoneCh <- "done"
							return
						}
						if sk != nil {
							skJsonBytes, err := json.Marshal(sk)
							if err != nil {
								t.Error(err)
								return
							}
							path, err := os.Getwd()
							if err != nil {
								t.Error(err)
								return
							}
							fileName := fmt.Sprintf("sk_%d.json", pIndex)
							fullName := fmt.Sprintf("%s/%s", path, fileName)
							err = os.WriteFile(fullName, skJsonBytes, os.ModePerm)
							if err != nil {
								t.Error(err)
								return
							}
							t.Logf("SK: %v", sk)
							oDoneCh <- "Done"
							nDoneCh <- "done"
						}
					}()
				case <-nDoneCh:
					break Loop
				default:
					continue Loop
				}
			}
		}(i)
	}
	wg.Wait()
}
