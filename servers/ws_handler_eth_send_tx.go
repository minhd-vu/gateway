package servers

import (
	"context"
	"fmt"

	"github.com/bloXroute-Labs/gateway/v2/connections"
	"github.com/bloXroute-Labs/gateway/v2/jsonrpc"
	"github.com/bloXroute-Labs/gateway/v2/utils"
	"github.com/sourcegraph/jsonrpc2"
)

func (h *handlerObj) handleRPCEthSendTx(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request, rpcParams []interface{}) {
	if len(rpcParams) != 1 {
		err := fmt.Sprintf("unable to process %v RPC request: expected 1, got %d", jsonrpc.RPCEthSendRawTransaction, len(rpcParams))
		SendErrorMsg(ctx, jsonrpc.InvalidParams, err, conn, req.ID)
		return
	}

	rawTxParam := rpcParams[0]
	var rawTxStr string
	switch rawTxParam.(type) {
	case string:
		rawTxStr = rawTxParam.(string)
		if len(rawTxStr) < 3 {
			err := fmt.Sprintf("unable to process %s RPC request: raw transaction string is too short", jsonrpc.RPCEthSendRawTransaction)
			SendErrorMsg(ctx, jsonrpc.InvalidParams, err, conn, req.ID)
			return
		}
		if rawTxStr[0:2] != "0x" {
			err := fmt.Sprintf("unable to process %s RPC request: expected raw transaction string to begin with '0x'", jsonrpc.RPCEthSendRawTransaction)
			SendErrorMsg(ctx, jsonrpc.InvalidParams, err, conn, req.ID)
			return
		}
		rawTxStr = rawTxStr[2:]
	default:
		err := fmt.Sprintf("unable to process %s RPC request: param must be a raw transaction string", jsonrpc.RPCEthSendRawTransaction)
		SendErrorMsg(ctx, jsonrpc.InvalidParams, err, conn, req.ID)
		return
	}

	reqWS := connections.NewRPCConn(h.connectionAccount.AccountID, h.remoteAddress, h.FeedManager.networkNum, utils.Websocket)
	txHash, ok, err := HandleSingleTransaction(h.FeedManager, rawTxStr, nil, reqWS, false, false,
		false, false, 0, nil, nil)
	if err != nil {
		SendErrorMsg(ctx, jsonrpc.InvalidParams, err.Error(), conn, req.ID)
	}
	if !ok {
		return
	}

	if err = conn.Reply(ctx, req.ID, "0x"+txHash); err != nil {
		h.log.Errorf("error replying to %v, method %v: %v", h.remoteAddress, req.Method, err)
	}
}
