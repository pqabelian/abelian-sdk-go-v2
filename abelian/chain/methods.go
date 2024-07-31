package chain

func (client *Client) GetChainInfo() (res *ChainInfo, err error) {
	err = client.Do("getinfo", nil, &res)
	return res, err
}

func (client *Client) GetRawMempool() (res []string, err error) {
	err = client.Do("getrawmempool", []interface{}{false}, &res)
	return res, err
}
func (client *Client) GetBlockHash(height int64) (res string, err error) {
	err = client.Do("getblockhash", []interface{}{height}, &res)
	return res, err
}
func (client *Client) GetBlock(blockID string) (res *Block, err error) {
	err = client.Do("getblockabe", []interface{}{blockID, 1}, &res)
	return res, err
}
func (client *Client) GetBlockBytes(blockID string) (res []byte, err error) {
	err = client.Do("getblockabe", []interface{}{blockID, 0}, &res)
	return res, err
}

func (client *Client) GetTxBytes(txID string) (res []byte, err error) {
	err = client.Do("getrawtransaction", []interface{}{txID, false}, &res)
	return res, err
}

func (client *Client) GetRawTx(txID string) (res *Tx, err error) {
	err = client.Do("getrawtransaction", []interface{}{txID, true}, &res)
	return res, err
}

func (client *Client) GetBlockByHeight(height int64) (res *Block, err error) {
	blockID, err := client.GetBlockHash(height)
	if err != nil {
		return nil, err
	}

	return client.GetBlock(blockID)
}

func (client *Client) SendRawTx(txID string) (res string, err error) {
	err = client.Do("sendrawtransactionabe", []interface{}{txID}, &res)
	return res, err
}
