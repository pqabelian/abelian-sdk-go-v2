package database

import (
	"fmt"
	"github.com/pqabelian/abelian-sdk-go-v2/abelian"
	"strings"
)

type Ring struct {
	ID           int64
	RingID       string
	RingVersion  uint32
	RingHeight   int64
	RingBlockIDs []string
	RingSize     int8
	IsCoinbase   bool
	Coins        []*abelian.Coin
}

func InsertRing(
	ringID string,
	ringVersion uint32,
	ringHeight int64,
	ringBlockIDs []string,
	ringSize int8,
	isCoinbase bool) (int64, error) {
	// check exist firstly
	exist, err := db.Query(`SELECT id FROM ring WHERE ring_id = ? AND ring_version = ?`, ringID, ringVersion)
	if err != nil {
		return -1, err
	}
	defer exist.Close()
	if exist.Next() {
		var id int64
		err := exist.Scan(&id)
		if err != nil {
			return -1, err
		}
		return id, err
	}
	stmt, err := db.Prepare(`INSERT INTO ring (ring_id,ring_version,ring_height,ring_block_ids ,ring_size,from_coinbase) 
									VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return -1, err
	}
	result, err := stmt.Exec(
		ringID,
		ringVersion,
		ringHeight,
		strings.Join(ringBlockIDs, ","),
		ringSize,
		isCoinbase)
	if err != nil {
		return -1, err
	}
	return result.LastInsertId()
}
func LoadRing(ringID string) (*Ring, error) {
	rows, err := db.Query(`
SELECT ID,ring_id,ring_version,ring_height,ring_block_ids ,ring_size,from_coinbase 
FROM ring  
WHERE ring_id=?`, ringID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ring := &Ring{}
	for rows.Next() {
		var id int64
		var ringID string
		var ringVersion uint32
		var ringHeight int64
		var ringBlockIDs string
		var ringSize int8
		var isCoinbase bool

		err = rows.Scan(
			&id,
			&ringID,
			&ringVersion,
			&ringHeight,
			&ringBlockIDs,
			&ringSize,
			&isCoinbase)
		if err != nil {
			return nil, err
		}

		ring.ID = id
		ring.RingID = ringID
		ring.RingVersion = ringVersion
		ring.RingHeight = ringHeight
		ring.RingBlockIDs = strings.Split(ringBlockIDs, ",")
		ring.RingSize = ringSize
		ring.IsCoinbase = isCoinbase
	}

	coins, err := loadCoinByRingID(ringID)
	if err != nil {
		return nil, err
	}
	// assert
	if len(coins) != int(ring.RingSize) {
		return nil, fmt.Errorf("ring size not match")
	}
	ring.Coins = make([]*abelian.Coin, len(coins))
	for i := 0; i < len(coins); i++ {
		ring.Coins[i] = coins[i].Coin
	}
	return ring, err
}
