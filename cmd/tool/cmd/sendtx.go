/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

var (
	// tps number of sended tx every second
	tps int

	// debug print more info
	debug bool

	// send tx mode
	// mode 1: send tx and check the txs periodically. add them back to balances after confirmed
	// mode 2: send tx with amount 1 and don't care about new balances
	// mode 3: split tx to 3 parts. send tx with full amount and replace old balances with new balances
	// mode 4: send tx to different shard
	// mode 5: send tx to different shards and same shard randomly
	// mode 6: send tx to different shards by cross number
	mode int

	// wg sync signal
	wg = sync.WaitGroup{}

	// receivers address
	receiversAddress map[uint][]KeyInfo

	// isRandom default false
	isRandom bool

	// cross is number of the crossing shard txs
	cross uint
)

type balance struct {
	address    *common.Address
	privateKey *ecdsa.PrivateKey
	amount     int
	shard      uint
	nonce      uint64
	tx         *common.Hash
	packed     bool
}

var sendTxCmd = &cobra.Command{
	Use:   "sendtx",
	Short: "send tx peroidly",
	Long: `For example:
	 tool.exe sendtx`,
	Run: func(cmd *cobra.Command, args []string) {
		initClient()
		balanceList := initAccount(threads)
		// receiversAddress init
		receiversAddress = make(map[uint][]KeyInfo)
		if receivers == "" {
			isRandom = true
		} else {
			isRandom = false
			initToAccount()
		}

		fmt.Println("use mode ", mode)
		fmt.Println("threads", threads)
		fmt.Println("is send to random address", isRandom)
		fmt.Println("total balance ", len(balanceList))
		balances := newBalancesList(balanceList, threads, true)

		for i := 0; i < threads; i++ {
			wg.Add(1)
			go StartSend(balances[i], i)
		}

		wg.Wait()
	},
}

// initToAccount init to accounts which are used to send tx
func initToAccount() {
	data, err := ioutil.ReadFile(receivers)
	if err != nil {
		panic(fmt.Sprintf("failed to read receivers file %s", err))
	}

	if err = json.Unmarshal(data, &receiversAddress); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal %s", err))
	}

}

// StartSend start send tx by specific thread and mode
func StartSend(balanceList []*balance, threadNum int) {
	lock := &sync.Mutex{}

	switch mode {
	case 1:
		go loopSendMode(balanceList, lock, threadNum)
		wg.Add(1)
		go loopCheckMode1(balanceList, lock)

	case 3:
		go loopSendMode3(balanceList)

	case 2, 4, 5, 6:
		go loopSendMode(balanceList, lock, threadNum)

	default:
		fmt.Printf("Invalid mode %d, supporting 1, 2, 3, 4, 5, 6", mode)
		break
	}
}

var tpsStartTime time.Time
var tpsCount = 0

func loopSendMode3(balanceList []*balance) {
	defer wg.Done()

	balances := newBalancesList(balanceList, 3, false)
	nextBalances := newBalancesList(balanceList, 3, true)

	tpsStartTime = time.Now()
	// send tx periodically
	for {
		SendMode3(balances[0], nextBalances[0])
		SendMode3(balances[1], nextBalances[1])
		SendMode3(balances[2], nextBalances[2])
	}
}

func newBalancesList(balanceList []*balance, splitNum int, copyValue bool) [][]*balance {
	balances := make([][]*balance, splitNum)
	unit := len(balanceList) / splitNum

	for i := 0; i < splitNum; i++ {
		var start = unit * i
		var end = unit * (i + 1)
		if i == splitNum-1 {
			end = len(balanceList)
		}

		balances[i] = make([]*balance, end-start)

		if copyValue {
			fmt.Printf("balance %d length %d\n", i, end-start)
			copy(balances[i], balanceList[start:end])
		}
	}

	return balances
}

// SendMode3 loop generate tx by balances, send the tx and update balances
func SendMode3(current []*balance, next []*balance) {
	copy(current, next)
	for i, b := range current {
		newBalance := send(b)
		if debug {
			fmt.Printf("send tx %s, account %s, nonce %d\n", newBalance.tx.Hex(), b.address.Hex(), b.nonce-1)
		}

		next[i] = newBalance

		tpsCount++
		if tpsCount == tps {
			fmt.Printf("send txs %d, [%d]\n", tpsCount, i)
			elapse := time.Now().Sub(tpsStartTime)
			if elapse < time.Second {
				time.Sleep(time.Second - elapse)
			}

			tpsCount = 0
			tpsStartTime = time.Now()
		}
	}
}

var txCh = make(chan *balance, 100000)

func loopSendMode(balanceList []*balance, lock *sync.Mutex, threadNum int) {
	defer wg.Done()

	count := 0
	tpsStartTime = time.Now()

	// send tx periodically
	for len(balanceList) > 0 {
		lock.Lock()
		copyBalances := make([]*balance, len(balanceList))
		copy(copyBalances, balanceList)
		fmt.Printf("balance total length %d at thread %d\n", len(balanceList), threadNum)
		lock.Unlock()

		for _, b := range copyBalances {
			switch mode {

			// 1 is used to send tx and print txs which are in pending and and block
			case 1:
				newBalance := send(b)
				if newBalance.amount > 0 {
					txCh <- newBalance
				}

				// 2 is used to senx tx in same shard
			case 2:
				send(b)

				// 4 is just used to send tx in different shards
			case 4:
				if common.ShardCount > 1 {
					sendDifferentShard(b)
				} else {
					panic(fmt.Sprintf("Failed to send tx in different shards, common shardcount is: %d", common.ShardCount))
				}

				// 5 is used to send tx in same shards or different shards randomly
			case 5:
				sendDifferentOrSameShard(b)

				// 6 is used to send tx in same shards or different shards, different shard tx number is limited by cross parameter
			case 6:
				if count < int(cross) {
					sendDifferentShard(b)
				} else {
					send(b)
				}

			default:
				send(b)
			}

			count++
			if count == tps {
				elapse := time.Now().Sub(tpsStartTime)
				fmt.Printf("from shard is %d sending txs %d at thread %d during %.2fs\n", b.shard, count, threadNum, elapse.Seconds())
				if elapse < time.Second {
					time.Sleep(time.Second - elapse)
				}

				count = 0
				tpsStartTime = time.Now()
			}
		}

		lock.Lock()
		nextBalanceList := make([]*balance, 0)
		for _, b := range balanceList {
			if b.amount > 0 {
				nextBalanceList = append(nextBalanceList, b)
			}
		}
		balanceList = nextBalanceList
		lock.Unlock()
	}
}

func loopCheckMode1(balanceList []*balance, lock *sync.Mutex) {
	defer wg.Done()
	toPackedBalanceList := make([]*balance, 0)
	toConfirmBalanceList := make(map[time.Time][]*balance)

	var confirmTime = 2 * time.Minute
	checkPack := time.NewTicker(30 * time.Second)
	confirm := time.NewTicker(30 * time.Second)
	for {
		select {
		case b := <-txCh:
			toPackedBalanceList = append(toPackedBalanceList, b)
		case <-checkPack.C:
			included, pending := getIncludedAndPendingBalance(toPackedBalanceList)
			toPackedBalanceList = pending

			fmt.Printf("to packed balance: %d, new: %d\n", len(toPackedBalanceList), len(pending))
			toConfirmBalanceList[time.Now()] = included
			toPackedBalanceList = pending
		case <-confirm.C:
			for key, value := range toConfirmBalanceList {
				duration := time.Now().Sub(key)
				if duration > confirmTime {

					lock.Lock()
					balanceList = append(balanceList, value...)
					fmt.Printf("add confirmed balance %d, new: %d\n", len(value), len(balanceList))
					lock.Unlock()

					delete(toConfirmBalanceList, key)
				}
			}
		}
	}
}

func getIncludedAndPendingBalance(balances []*balance) ([]*balance, []*balance) {
	include := make([]*balance, 0)
	pending := make([]*balance, 0)
	for _, b := range balances {
		if b.tx == nil {
			continue
		}

		result := getTx(*b.address, *b.tx)
		if len(result) > 0 {
			if result["status"] == "block" {
				include = append(include, b)
			} else if result["status"] == "pool" {
				pending = append(pending, b)
			}

			if debug {
				fmt.Printf("got tx success %s from %s nonce %.0f status %s amount %.0f\n", b.tx.Hex(), result["from"],
					result["accountNonce"], result["status"], result["amount"])
			}
		}
	}

	return include, pending
}

func getTx(address common.Address, hash common.Hash) map[string]interface{} {
	client := getClient(address)

	result, err := util.GetTransactionByHash(client, hash.Hex())
	if err != nil {
		fmt.Println("failed to get tx ", err, " tx hash ", hash.Hex())
		return result
	}

	return result
}

func send(b *balance) *balance {
	var amount = 1
	if mode == 1 {
		amount = rand.Intn(b.amount) // for test, amount will always keep in int value.
	} else if mode == 3 {
		amount = b.amount
	}

	return sendtx(b, amount, b.address.Shard())
}

// getRandomShard get random sahrd
func getRandomShard() uint {
	rand.Seed(time.Now().UnixNano())

	return uint(rand.Int31n(common.ShardCount) + 1)
}

// sendDifferentOrSameShard tx is in different shard or in same shard
func sendDifferentOrSameShard(b *balance) *balance {
	var amount = 1
	shard := getRandomShard()

	return sendtx(b, amount, shard)
}

// sendDifferentShard is used to send tx from different shard in mode 4
func sendDifferentShard(b *balance) *balance {
	var amount = 1
	var shard uint
	if common.ShardCount > 1 {
		for {
			shard = getRandomShard()
			if shard != b.address.Shard() {
				break
			}
		}

		return sendtx(b, amount, shard)
	}

	return nil
}

func sendtx(b *balance, amount int, shard uint) *balance {
	var addr *common.Address
	var privateKey *ecdsa.PrivateKey

	if isRandom {
		addr, privateKey = crypto.MustGenerateShardKeyPair(shard)

	} else {
		data := receiversAddress[shard]
		index := rand.Intn(len(data))
		addr = data[index].Addr
		key, err := crypto.LoadECDSAFromString(data[index].PrivateKey)
		if err != nil {
			panic(fmt.Sprintf("Failed to load private key from string %s", err))
		}

		privateKey = key
	}

	newBalance := &balance{
		address:    addr,
		privateKey: privateKey,
		amount:     amount,
		shard:      shard,
		nonce:      0,
		packed:     false,
	}

	value := big.NewInt(int64(amount))
	value.Mul(value, common.SeeleToFan)

	client := getRandClient()
	tx, err := util.GenerateTx(b.privateKey, *addr, value, big.NewInt(1), 0, b.nonce, nil)
	if err != nil {
		return newBalance
	}

	ok, err := util.SendTx(client, tx)
	if !ok || err != nil {
		return newBalance
	}

	// update balance by transaction amount and update nonce
	b.nonce++
	b.amount -= amount
	newBalance.tx = &tx.Hash

	return newBalance
}

func getRandClient() *rpc.Client {
	if len(clientList) == 0 {
		panic("no client found")
	}

	index := rand.Intn(len(clientList))

	count := 0
	for _, v := range clientList {
		if count == index {
			return v
		}

		count++
	}

	return nil
}

func initAccount(threads int) []*balance {
	keys, err := ioutil.ReadFile(keyFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read key file %s", err))
	}

	keyList := strings.Split(string(keys), "\r\n")
	unit := len(keyList) / threads

	wg := &sync.WaitGroup{}
	balanceList := make([]*balance, len(keyList))
	for i := 0; i < threads; i++ {
		end := (i + 1) * unit
		if i == threads-1 {
			end = len(keyList)
		}

		wg.Add(1)
		go initBalance(balanceList, keyList, i*unit, end, wg)
	}

	wg.Wait()

	result := make([]*balance, 0)
	for _, b := range balanceList {
		if b != nil && b.amount > 0 {
			result = append(result, b)
		}
	}

	return result
}

func initBalance(balanceList []*balance, keyList []string, start int, end int, wg *sync.WaitGroup) {
	defer wg.Done()

	// init balance and nonce
	for i := start; i < end; i++ {
		hex := keyList[i]
		if hex == "" {
			continue
		}

		key, err := crypto.LoadECDSAFromString(hex)
		if err != nil {
			panic(fmt.Sprintf("failed to load key %s", err))
		}

		addr := crypto.GetAddress(&key.PublicKey)
		// skip address that don't find the same shard client
		if _, ok := clientList[addr.Shard()]; !ok {
			continue
		}

		amount, ok := getBalance(*addr, "", -1)
		if !ok {
			continue
		}

		b := &balance{
			address:    addr,
			privateKey: key,
			amount:     amount,
			shard:      addr.Shard(),
			packed:     false,
		}

		fmt.Printf("%s balance is %d\n", b.address.Hex(), b.amount)

		if b.amount > 0 {
			b.nonce = getNonce(*b.address, "", -1)
			balanceList[i] = b
		}
	}
}

func getBalance(address common.Address, hexHash string, height int64) (int, bool) {
	client := getClient(address)

	var result api.GetBalanceResponse
	if err := client.Call(&result, "seele_getBalance", address, hexHash, height); err != nil {
		panic(fmt.Sprintf("failed to get the balance: %s\n", err))
	}

	return int(result.Balance.Div(result.Balance, common.SeeleToFan).Uint64()), true
}

func getClient(address common.Address) *rpc.Client {
	shard := address.Shard()
	client := clientList[shard]
	if client == nil {
		panic(fmt.Sprintf("not found client in shard %d", shard))
	}

	return client
}

// getNonce get current nonce
func getNonce(address common.Address, hexHash string, height int64) uint64 {
	client := getClient(address)

	//get current nonce
	nonce, err := util.GetAccountNonce(client, address, hexHash, height)
	if err != nil {
		panic(err)
	}

	return nonce
}

func getShard(client *rpc.Client) uint {
	info, err := util.GetInfo(client)
	if err != nil {
		panic(fmt.Sprintf("failed to get the balance: %s\n", err.Error()))
	}

	return info.Coinbase.Shard() // @TODO need refine this code, get shard info straight
}

func init() {
	rootCmd.AddCommand(sendTxCmd)

	sendTxCmd.Flags().StringVarP(&keyFile, "keyfile", "f", "keystore.txt", "key store file")
	sendTxCmd.Flags().StringVarP(&receivers, "receiver", "r", "", "receiver address file")
	sendTxCmd.Flags().IntVarP(&tps, "tps", "", 3, "target tps to send transaction")
	sendTxCmd.Flags().BoolVarP(&debug, "debug", "d", false, "whether print more debug info")
	sendTxCmd.Flags().IntVarP(&mode, "mode", "m", 1, "send tx mode")
	sendTxCmd.Flags().IntVarP(&threads, "threads", "t", 1, "send tx threads")
	sendTxCmd.Flags().UintVarP(&cross, "cross", "c", 0, "cross shard txs number")
}
