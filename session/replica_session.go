// Copyright (c) 2017, Xiaomi, Inc.  All rights reserved.
// This source code is licensed under the Apache License Version 2.0, which
// can be found in the LICENSE file in the root directory of this source tree.

package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/XiaoMi/pegasus-go-client/idl/base"
	"github.com/XiaoMi/pegasus-go-client/idl/rrdb"
)

// ReplicaSession represents the network session between client and
// replica server.
type ReplicaSession struct {
	*nodeSession
}

func newReplicaSession(addr string) *ReplicaSession {
	return &ReplicaSession{
		nodeSession: newNodeSession(addr, kNodeTypeReplica),
	}
}

func (rs *ReplicaSession) Get(ctx context.Context, gpid *base.Gpid, key *base.Blob) (*rrdb.ReadResponse, error) {
	args := &rrdb.RrdbGetArgs{Key: key}
	result, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_GET")
	if err != nil {
		return nil, err
	}

	ret, _ := result.(*rrdb.RrdbGetResult)
	return ret.GetSuccess(), nil
}

func (rs *ReplicaSession) Put(ctx context.Context, gpid *base.Gpid, key *base.Blob, value *base.Blob) (*rrdb.UpdateResponse, error) {
	update := &rrdb.UpdateRequest{Key: key, Value: value}
	args := &rrdb.RrdbPutArgs{Update: update}

	result, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_PUT")
	if err != nil {
		return nil, err
	}

	ret, _ := result.(*rrdb.RrdbPutResult)
	return ret.GetSuccess(), nil
}

func (rs *ReplicaSession) Del(ctx context.Context, gpid *base.Gpid, key *base.Blob) (*rrdb.UpdateResponse, error) {
	args := &rrdb.RrdbRemoveArgs{Key: key}
	result, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_REMOVE")
	if err != nil {
		return nil, err
	}

	ret, _ := result.(*rrdb.RrdbRemoveResult)
	return ret.GetSuccess(), nil
}

func (rs *ReplicaSession) MultiGet(ctx context.Context, gpid *base.Gpid, request *rrdb.MultiGetRequest) (*rrdb.MultiGetResponse, error) {
	args := &rrdb.RrdbMultiGetArgs{Request: request}
	result, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_MULTI_GET")
	if err != nil {
		return nil, err
	}

	ret, _ := result.(*rrdb.RrdbMultiGetResult)
	return ret.GetSuccess(), nil
}

func (rs *ReplicaSession) MultiSet(ctx context.Context, gpid *base.Gpid, request *rrdb.MultiPutRequest) (*rrdb.UpdateResponse, error) {
	args := &rrdb.RrdbMultiPutArgs{Request: request}
	result, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_MULTI_PUT")
	if err != nil {
		return nil, err
	}

	ret, _ := result.(*rrdb.RrdbMultiPutResult)
	return ret.GetSuccess(), nil
}

func (rs *ReplicaSession) MultiDelete(ctx context.Context, gpid *base.Gpid, request *rrdb.MultiRemoveRequest) (*rrdb.MultiRemoveResponse, error) {
	args := &rrdb.RrdbMultiRemoveArgs{Request: request}
	result, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_MULTI_REMOVE")
	if err != nil {
		return nil, err
	}

	ret, _ := result.(*rrdb.RrdbMultiRemoveResult)
	return ret.GetSuccess(), nil
}

func (rs *ReplicaSession) TTL(ctx context.Context, gpid *base.Gpid, key *base.Blob) (*rrdb.TTLResponse, error) {
	args := &rrdb.RrdbTTLArgs{Key: key}
	result, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_TTL")
	if err != nil {
		return nil, err
	}

	ret, _ := result.(*rrdb.RrdbTTLResult)
	return ret.GetSuccess(), nil
}

func (rs *ReplicaSession) GetScanner(ctx context.Context, gpid *base.Gpid, request *rrdb.GetScannerRequest) (*rrdb.ScanResponse, error) {
	args := &rrdb.RrdbGetScannerArgs{Request: request}
	result, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_GET_SCANNER")
	if err != nil {
		return nil, err
	}

	ret, _ := result.(*rrdb.RrdbGetScannerResult)
	return ret.GetSuccess(), nil
}

func (rs *ReplicaSession) Scan(ctx context.Context, gpid *base.Gpid, request *rrdb.ScanRequest) (*rrdb.ScanResponse, error) {
	args := &rrdb.RrdbScanArgs{Request: request}
	result, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_SCAN")
	if err != nil {
		return nil, err
	}

	ret, _ := result.(*rrdb.RrdbScanResult)
	return ret.GetSuccess(), nil
}

func (rs *ReplicaSession) ClearScanner(ctx context.Context, gpid *base.Gpid, contextId int64) error {
	args := &rrdb.RrdbClearScannerArgs{ContextID: contextId}
	_, err := rs.callWithGpid(ctx, gpid, args, "RPC_RRDB_RRDB_CLEAR_SCANNER")
	if err != nil {
		return err
	}

	return nil
}

func (rs *ReplicaSession) String() string {
	return fmt.Sprintf("replica(%s)", rs.addr)
}

// ReplicaManager manages the pool of sessions to replica servers, so that
// different tables that locate on the same replica server can share one
// ReplicaSession, without the effort of creating a new connection.
type ReplicaManager struct {
	//	rpc address -> replica
	replicas map[string]*ReplicaSession
	sync.RWMutex
}

// Create a new session to the replica server if no existing one.
func (rm *ReplicaManager) GetReplica(addr string) *ReplicaSession {
	rm.Lock()
	defer rm.Unlock()

	if _, ok := rm.replicas[addr]; !ok {
		rm.replicas[addr] = newReplicaSession(addr)
	}
	return rm.replicas[addr]
}

func NewReplicaManager() *ReplicaManager {
	return &ReplicaManager{
		replicas: make(map[string]*ReplicaSession),
	}
}

func (rm *ReplicaManager) Close() error {
	rm.Lock()
	defer rm.Unlock()

	for _, r := range rm.replicas {
		<-r.Close()
	}
	return nil
}

func (rm *ReplicaManager) ReplicaCount() int {
	rm.RLock()
	defer rm.RUnlock()

	return len(rm.replicas)
}
